// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   demo 装配主程序 — T-001~T-005 五大块全链路跑通
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 04:02:00
// +----------------------------------------------------------------------

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/crypto"
	"github.com/benxin_dev/benxinadminpro-server/drivers/storage"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/benxin_dev/benxinadminpro-server/system"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// ---- 1. 加载配置 ----
	cfg := loadConfig()

	// ---- 2. 连接依赖 ----
	db := mustConnectMySQL(cfg)
	rdb := mustConnectRedis(cfg)

	// ---- 3. 迁移执行器建表 ----
	migrationsDir := findMigrationsDir()
	migrator := system.NewMigrator(db, cfg.TablePrefix, migrationsDir)
	if err := migrator.Up(context.Background()); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	slog.Info("migrations applied")

	// ---- 4. 装配各模块 ----

	// errcode + response Registry
	ecReg, err := errcode.NewRegistry(cfg.SegmentBase)
	if err != nil {
		log.Fatalf("errcode: %v", err)
	}
	if err := response.InitFromErrcode(ecReg.AllSpecs()); err != nil {
		log.Fatalf("response init: %v", err)
	}

	// crypto 密钥
	aesKey := mustDecodeB64(cfg.AESKeyB64, 32, "crypto.aes_key_b64")
	hmacKey := mustDecodeB64(cfg.HMACKeyB64, 0, "crypto.hmac_key_b64")
	gcmKey := mustDecodeB64(cfg.GCMKeyB64, 32, "gcm.key_b64")

	// auth
	hasher, _ := auth.NewPasswordHasher(cfg.Argon2Params)
	tokenStore := auth.NewRedisBlacklistStore(rdb)
	tokenSvc, _ := auth.NewTokenService(auth.Config{
		Issuer: cfg.JWTIssuer, AccessSecret: cfg.JWTAccessSecret, RefreshSecret: cfg.JWTRefreshSecret,
		AccessTTLSeconds: cfg.JWTAccessTTL, RefreshTTLSeconds: cfg.JWTRefreshTTL,
		RefreshRotate: true, RedisKeyPrefix: cfg.RedisPrefix,
	}, tokenStore, ecReg)

	captchaStore := auth.NewRedisCaptchaStore(rdb)
	captchaSvc := auth.NewCaptchaService(captchaStore, auth.CaptchaConfig{
		Enabled: true, TTLSeconds: 120, Length: 4,
	}, cfg.RedisPrefix)

	lockoutStore := auth.NewRedisLockoutStore(rdb)
	lockoutSvc, _ := auth.NewLockoutService(lockoutStore, auth.LockoutConfig{
		CaptchaThreshold: 3, LockThreshold: 5, FailWindowSeconds: 900, LockDurationSeconds: 900,
	}, cfg.RedisPrefix)

	// rbac
	userProvider := rbac.NewGormUserProvider(db)
	loginLogger := system.NewGormLoginLogger(db)

	authSvc, _ := auth.NewAuthService(auth.AuthServiceDeps{
		Tokens: tokenSvc, Users: userProvider, Hasher: hasher,
		CaptchaSvc: captchaSvc, LockoutSvc: lockoutSvc, Errs: ecReg,
		Argon2Params: cfg.Argon2Params, LoginLogger: loginLogger,
	})

	// hashid
	hashidHasher, err := rbac.NewHasher(rbac.HashidConfig{Salt: cfg.HashidSalt, MinLength: 8})
	if err != nil {
		log.Fatalf("hashid: %v", err)
	}

	// casbin enforcer
	enforcer, err := rbac.NewEnforcer(db, findModelConfPath(), cfg.TablePrefix)
	if err != nil {
		log.Fatalf("casbin: %v", err)
	}
	policySync := rbac.NewPolicySync(enforcer, db)

	// scope resolver
	scopeResolver := rbac.NewGormScopeResolver(db, cfg.SuperAdminRoles)

	// system services
	dictSvc := system.NewDictService(db, ecReg)
	configSvc := system.NewConfigService(db, ecReg)
	logSvc := system.NewLogService(db)
	operSink := &system.GormOperLogSink{DB: db}

	// file
	localDriver, _ := storage.NewLocalDriver(cfg.FileRootDir, cfg.FileURLPrefix)
	fileSvc := system.NewFileService(db, localDriver, storage.UploadConfig{
		MaxSizeBytes: cfg.FileMaxSize, AllowedExts: cfg.FileAllowedExts,
	}, ecReg, "local")

	// config center
	redisCache := system.NewRedisConfigCache(rdb)
	redisPub := system.NewRedisPublisher(rdb, cfg.RedisPrefix+":config:changed")
	configCenter := system.NewConfigCenter(db, redisCache, redisPub, system.CenterConfig{
		CacheTTL: 5 * time.Minute, RedisKeyPrefix: cfg.RedisPrefix, GCMKey: gcmKey,
	})

	// Pub/Sub 订阅
	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	redisSub := system.NewRedisSubscriber(rdb, cfg.RedisPrefix+":config:changed")
	redisSub.Subscribe(subCtx, configCenter.OnChange)

	// ---- 5. 装配自检 ----
	mustNotNil("hashid hasher", hashidHasher)
	mustNotNil("token service", tokenSvc)
	mustNotNil("enforcer", enforcer)
	mustNotNil("config center", configCenter)
	slog.Info("assembly self-check passed")

	// ---- 6. 种子数据 ----
	seed(db, hasher, policySync, cfg)

	// ---- 7. 路由总装 ----
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 公共路由（无鉴权）
	authHandler := auth.NewHandler(authSvc, ecReg)
	authHandler.RegisterRoutes(&r.RouterGroup)

	// 受保护路由组
	protected := r.Group("")
	protected.Use(rbac.JWTAuth(tokenSvc, ecReg))

	// Authz enforcer
	subjectFn := func(c *gin.Context) string {
		claims := rbac.GetClaims(c)
		if claims != nil {
			return claims.Subject
		}
		return ""
	}
	authzEnforcer := rbac.NewAuthzEnforcer(enforcer, rbac.AuthzConfig{
		SuperAdminRoles: cfg.SuperAdminRoles,
	}, subjectFn, ecReg)
	_ = authzEnforcer // 各 handler 路由上用 RequirePerm（包级函数设 context value）

	// 操作日志中间件
	protected.Use(system.OperLog(system.OperLogConfig{
		Enabled: true, ExcludeMethods: []string{"GET"}, BodyMaxBytes: 1024,
	}, operSink, subjectFn))

	// 各模块 handler
	userSvc := rbac.NewUserService(db, hasher, ecReg)
	userSvc.SetPolicySync(policySync)
	userHandler := rbac.NewUserHandler(userSvc, ecReg, hashidHasher)
	userHandler.SetScopeResolver(scopeResolver)
	userHandler.RegisterRoutes(protected)

	deptSvc := rbac.NewDeptService(db, ecReg)
	rbac.NewDeptHandler(deptSvc, ecReg, hashidHasher).RegisterRoutes(protected)

	postSvc := rbac.NewPostService(db, ecReg)
	rbac.NewPostHandler(postSvc, ecReg, hashidHasher).RegisterRoutes(protected)

	roleSvc := rbac.NewRoleService(db, ecReg, policySync)
	rbac.NewRoleHandler(roleSvc, ecReg, hashidHasher).RegisterRoutes(protected)

	menuSvc := rbac.NewMenuService(db, ecReg)
	rbac.NewMenuHandler(menuSvc, ecReg, hashidHasher).RegisterRoutes(protected)

	rbac.NewAuthInfoHandler(menuSvc, userSvc, hashidHasher).RegisterRoutes(protected)

	sysHandler := system.NewHandler(dictSvc, configSvc, logSvc, ecReg)
	sysHandler.RegisterRoutes(protected)

	fileHandler := system.NewFileHandler(fileSvc, ecReg)
	fileHandler.RegisterRoutes(protected)

	// C 端信封示例（可选）
	cryptoMW := setupCryptoMiddleware(aesKey, hmacKey, ecReg, rdb, cfg.RedisPrefix)
	if cryptoMW != nil {
		cEnd := r.Group("/api/c")
		cEnd.Use(cryptoMW.Handler())
		cEnd.POST("/echo", func(c *gin.Context) {
			body, _ := c.GetRawData()
			c.JSON(200, gin.H{"echo": string(body)})
		})
	}

	// ---- 8. 启动 ----
	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("demo starting", slog.String("addr", addr))

	srv := &http.Server{Addr: addr, Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")
	subCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

	_ = configCenter // keep reference
}

// ---------------------------------------------------------------------------
// 配置
// ---------------------------------------------------------------------------

type demoConfig struct {
	TablePrefix    string
	RedisPrefix    string
	Port           int
	SegmentBase    int
	AESKeyB64      string
	HMACKeyB64     string
	GCMKeyB64      string
	JWTIssuer      string
	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL   int64
	JWTRefreshTTL  int64
	Argon2Params   auth.Argon2idParams
	SuperAdminRoles []string
	HashidSalt     string
	FileRootDir    string
	FileURLPrefix  string
	FileMaxSize    int64
	FileAllowedExts []string
	SeedAdminPwd   string
	MySQLDSN       string
	RedisAddr      string
	RedisDB        int
	RedisPassword  string
}

func loadConfig() demoConfig {
	v := viper.New()
	v.SetConfigName("config.local")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./examples/demo")
	v.SetEnvPrefix("DEMO")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("read config: %v (copy config.example.yaml to config.local.yaml)", err)
	}

	return demoConfig{
		TablePrefix:     v.GetString("app.table_prefix"),
		RedisPrefix:     v.GetString("app.redis_key_prefix"),
		Port:            v.GetInt("app.port"),
		SegmentBase:     v.GetInt("errcode.security_segment_base"),
		AESKeyB64:       v.GetString("crypto.aes_key_b64"),
		HMACKeyB64:      v.GetString("crypto.hmac_key_b64"),
		GCMKeyB64:       v.GetString("gcm.key_b64"),
		JWTIssuer:       v.GetString("jwt.issuer"),
		JWTAccessSecret: v.GetString("jwt.access_secret"),
		JWTRefreshSecret: v.GetString("jwt.refresh_secret"),
		JWTAccessTTL:    v.GetInt64("jwt.access_ttl_seconds"),
		JWTRefreshTTL:   v.GetInt64("jwt.refresh_ttl_seconds"),
		Argon2Params: auth.Argon2idParams{
			MemoryKiB: uint32(v.GetInt("auth.argon2id.memory_kib")),
			Iterations: uint32(v.GetInt("auth.argon2id.iterations")),
			Parallelism: uint8(v.GetInt("auth.argon2id.parallelism")),
			SaltLen: uint32(v.GetInt("auth.argon2id.salt_len")),
			KeyLen: uint32(v.GetInt("auth.argon2id.key_len")),
		},
		SuperAdminRoles: v.GetStringSlice("rbac.super_admin_roles"),
		HashidSalt:      v.GetString("hashid.salt"),
		FileRootDir:     v.GetString("file.local.root_dir"),
		FileURLPrefix:   v.GetString("file.local.url_prefix"),
		FileMaxSize:     v.GetInt64("file.max_size_bytes"),
		FileAllowedExts: v.GetStringSlice("file.allowed_exts"),
		SeedAdminPwd:    v.GetString("seed.admin_password"),
		MySQLDSN:        v.GetString("mysql.dsn"),
		RedisAddr:       v.GetString("redis.addr"),
		RedisDB:         v.GetInt("redis.db"),
		RedisPassword:   v.GetString("redis.password"),
	}
}

// ---------------------------------------------------------------------------
// 辅助
// ---------------------------------------------------------------------------

func mustConnectMySQL(cfg demoConfig) *gorm.DB {
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), rbac.NewDBConfig(cfg.TablePrefix))
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	return db
}

func mustConnectRedis(cfg demoConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr, DB: cfg.RedisDB, Password: cfg.RedisPassword,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis: %v", err)
	}
	return client
}

func mustDecodeB64(val string, expectLen int, name string) []byte {
	if val == "" {
		log.Fatalf("config %s must not be empty", name)
	}
	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		log.Fatalf("config %s invalid base64: %v", name, err)
	}
	if expectLen > 0 && len(decoded) != expectLen {
		log.Fatalf("config %s must decode to %d bytes, got %d", name, expectLen, len(decoded))
	}
	return decoded
}

func mustNotNil(name string, v any) {
	if v == nil {
		log.Fatalf("assembly self-check: %s is nil", name)
	}
}

func findMigrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "spec", "migrations")
}

func findModelConfPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "spec", "rbac", "model.conf")
}

func setupCryptoMiddleware(aesKey, hmacKey []byte, errs *errcode.Registry, rdb *redis.Client, prefix string) *crypto.Middleware {
	replayStore := crypto.NewRedisReplayStore(rdb)
	mw, err := crypto.NewMiddleware(crypto.Config{
		HeaderPrefix: "X-Ca-", AESKey: aesKey, HMACKey: hmacKey,
		ReplayWindowSeconds: 300, NonceTTLSeconds: 600, RedisKeyPrefix: prefix,
	}, replayStore, errs)
	if err != nil {
		slog.Warn("crypto middleware not configured", slog.String("error", err.Error()))
		return nil
	}
	return mw
}
