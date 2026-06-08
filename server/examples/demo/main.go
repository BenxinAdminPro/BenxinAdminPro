// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   demo 装配主程序 — T-001~T-005 五大块全链路跑通
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 04:02:00
// | @updated   2026-06-08 12:00:00  抽出 buildApp 供 e2e 集成测试；密钥/装配错误改返回 error
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
	"strings"
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

	// ---- 3. 装配（迁移建表 → 模块 DI → 种子 → 路由）----
	app, err := buildApp(cfg, db, rdb)
	if err != nil {
		log.Fatalf("build app: %v", err)
	}

	// ---- 4. 启动 ----
	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("demo starting", slog.String("addr", addr))

	srv := &http.Server{Addr: addr, Handler: app.handler}
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
	app.subCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// ---------------------------------------------------------------------------
// 装配（可测）
// ---------------------------------------------------------------------------

// demoApp 持有装配产物与生命周期句柄，供 main 与 e2e 测试共用。
type demoApp struct {
	handler      http.Handler
	db           *gorm.DB
	rdb          *redis.Client
	configCenter *system.ConfigCenter
	subCancel    context.CancelFunc // 关闭 Pub/Sub 订阅协程
}

// buildApp 完成「迁移建表 → 模块装配(DI) → 装配自检 → 种子数据 → 路由总装」。
// 任一步失败返回 error（startup fail-fast）；连接好的 db/rdb 由调用方注入并管理生命周期。
func buildApp(cfg demoConfig, db *gorm.DB, rdb *redis.Client) (*demoApp, error) {
	// ---- 迁移执行器建表 ----
	migrator := system.NewMigrator(db, cfg.TablePrefix, findMigrationsDir())
	if err := migrator.Up(context.Background()); err != nil {
		return nil, fmt.Errorf("migration: %w", err)
	}
	slog.Info("migrations applied")

	// ---- errcode + response Registry ----
	ecReg, err := errcode.NewRegistry(cfg.SegmentBase)
	if err != nil {
		return nil, fmt.Errorf("errcode: %w", err)
	}
	if err := response.InitFromErrcode(ecReg.AllSpecs()); err != nil {
		return nil, fmt.Errorf("response init: %w", err)
	}

	// ---- crypto 密钥 ----
	aesKey, err := decodeB64(cfg.AESKeyB64, 32, "crypto.aes_key_b64")
	if err != nil {
		return nil, err
	}
	hmacKey, err := decodeB64(cfg.HMACKeyB64, 0, "crypto.hmac_key_b64")
	if err != nil {
		return nil, err
	}
	gcmKey, err := decodeB64(cfg.GCMKeyB64, 32, "gcm.key_b64")
	if err != nil {
		return nil, err
	}

	// ---- auth ----
	hasher, err := auth.NewPasswordHasher(cfg.Argon2Params)
	if err != nil {
		return nil, fmt.Errorf("password hasher: %w", err)
	}
	tokenStore := auth.NewRedisBlacklistStore(rdb)
	tokenSvc, err := auth.NewTokenService(auth.Config{
		Issuer: cfg.JWTIssuer, AccessSecret: cfg.JWTAccessSecret, RefreshSecret: cfg.JWTRefreshSecret,
		AccessTTLSeconds: cfg.JWTAccessTTL, RefreshTTLSeconds: cfg.JWTRefreshTTL,
		RefreshRotate: true, RedisKeyPrefix: cfg.RedisPrefix,
	}, tokenStore, ecReg)
	if err != nil {
		return nil, fmt.Errorf("token service: %w", err)
	}

	captchaStore := auth.NewRedisCaptchaStore(rdb)
	captchaSvc := auth.NewCaptchaService(captchaStore, auth.CaptchaConfig{
		Enabled: true, TTLSeconds: 120, Length: 4,
	}, cfg.RedisPrefix)

	lockoutStore := auth.NewRedisLockoutStore(rdb)
	lockoutSvc, err := auth.NewLockoutService(lockoutStore, auth.LockoutConfig{
		CaptchaThreshold: 3, LockThreshold: 5, FailWindowSeconds: 900, LockDurationSeconds: 900,
	}, cfg.RedisPrefix)
	if err != nil {
		return nil, fmt.Errorf("lockout service: %w", err)
	}

	// ---- rbac ----
	userProvider := rbac.NewGormUserProvider(db)
	loginLogger := system.NewGormLoginLogger(db)

	authSvc, err := auth.NewAuthService(auth.AuthServiceDeps{
		Tokens: tokenSvc, Users: userProvider, Hasher: hasher,
		CaptchaSvc: captchaSvc, LockoutSvc: lockoutSvc, Errs: ecReg,
		Argon2Params: cfg.Argon2Params, LoginLogger: loginLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("auth service: %w", err)
	}

	hashidHasher, err := rbac.NewHasher(rbac.HashidConfig{Salt: cfg.HashidSalt, MinLength: 8})
	if err != nil {
		return nil, fmt.Errorf("hashid: %w", err)
	}

	enforcer, err := rbac.NewEnforcer(db, findModelConfPath(), cfg.TablePrefix)
	if err != nil {
		return nil, fmt.Errorf("casbin: %w", err)
	}
	policySync := rbac.NewPolicySync(enforcer, db)
	scopeResolver := rbac.NewGormScopeResolver(db, cfg.SuperAdminRoles)

	// ---- system services ----
	dictSvc := system.NewDictService(db, ecReg)
	configSvc := system.NewConfigService(db, ecReg)
	logSvc := system.NewLogService(db)
	operSink := &system.GormOperLogSink{DB: db}

	// ---- file ----
	localDriver, err := storage.NewLocalDriver(cfg.FileRootDir, cfg.FileURLPrefix)
	if err != nil {
		return nil, fmt.Errorf("local storage driver: %w", err)
	}
	fileSvc := system.NewFileService(db, localDriver, storage.UploadConfig{
		MaxSizeBytes: cfg.FileMaxSize, AllowedExts: cfg.FileAllowedExts,
	}, ecReg, "local")

	// ---- config center ----
	redisCache := system.NewRedisConfigCache(rdb)
	redisPub := system.NewRedisPublisher(rdb, cfg.RedisPrefix+":config:changed")
	configCenter := system.NewConfigCenter(db, redisCache, redisPub, system.CenterConfig{
		CacheTTL: 5 * time.Minute, RedisKeyPrefix: cfg.RedisPrefix, GCMKey: gcmKey,
	})

	// Pub/Sub 订阅（协程 + context 优雅退出）
	subCtx, subCancel := context.WithCancel(context.Background())
	redisSub := system.NewRedisSubscriber(rdb, cfg.RedisPrefix+":config:changed")
	redisSub.Subscribe(subCtx, configCenter.OnChange)

	// ---- 装配自检 ----
	for name, v := range map[string]any{
		"hashid hasher": hashidHasher, "token service": tokenSvc,
		"enforcer": enforcer, "config center": configCenter,
	} {
		if v == nil {
			subCancel()
			return nil, fmt.Errorf("assembly self-check: %s is nil", name)
		}
	}
	slog.Info("assembly self-check passed")

	// ---- 种子数据 ----
	if err := seed(db, hasher, policySync, cfg); err != nil {
		subCancel()
		return nil, fmt.Errorf("seed: %w", err)
	}

	// ---- 路由总装 ----
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 公共路由（无鉴权）
	auth.NewHandler(authSvc, ecReg).RegisterRoutes(&r.RouterGroup)

	// 受保护路由组
	protected := r.Group("")
	protected.Use(rbac.JWTAuth(tokenSvc, ecReg))

	subjectFn := func(c *gin.Context) string {
		claims := rbac.GetClaims(c)
		if claims != nil {
			return claims.Subject
		}
		return ""
	}
	// 鉴权 enforcer：注入各 RegisterRoutes，每条受保护路由走真 enforce（含超管短路）
	authzEnforcer := rbac.NewAuthzEnforcer(enforcer, rbac.AuthzConfig{
		SuperAdminRoles: cfg.SuperAdminRoles,
	}, subjectFn, ecReg)

	// 操作日志中间件
	protected.Use(system.OperLog(system.OperLogConfig{
		Enabled: true, ExcludeMethods: []string{"GET"}, BodyMaxBytes: 1024,
	}, operSink, subjectFn))

	// 各模块 handler（受保护路由链：JWTAuth → authz.RequirePerm(路由级 enforce) → OperLog → handler）
	userSvc := rbac.NewUserService(db, hasher, ecReg)
	userSvc.SetPolicySync(policySync)
	userHandler := rbac.NewUserHandler(userSvc, ecReg, hashidHasher)
	userHandler.SetScopeResolver(scopeResolver)
	userHandler.RegisterRoutes(protected, authzEnforcer)

	deptSvc := rbac.NewDeptService(db, ecReg)
	rbac.NewDeptHandler(deptSvc, ecReg, hashidHasher).RegisterRoutes(protected, authzEnforcer)

	postSvc := rbac.NewPostService(db, ecReg)
	rbac.NewPostHandler(postSvc, ecReg, hashidHasher).RegisterRoutes(protected, authzEnforcer)

	roleSvc := rbac.NewRoleService(db, ecReg, policySync)
	rbac.NewRoleHandler(roleSvc, ecReg, hashidHasher).RegisterRoutes(protected, authzEnforcer)

	menuSvc := rbac.NewMenuService(db, ecReg)
	rbac.NewMenuHandler(menuSvc, ecReg, hashidHasher).RegisterRoutes(protected, authzEnforcer)

	// auth_info（/sys/auth/menus、/sys/auth/perms）：返回当前用户自身菜单/权限，
	// 仅需 JWT 认证、按设计无需 perm code（非裸奔——已在 protected 组受 JWTAuth 保护）。
	rbac.NewAuthInfoHandler(menuSvc, userSvc, hashidHasher).RegisterRoutes(protected)

	system.NewHandler(dictSvc, configSvc, logSvc, ecReg).RegisterRoutes(protected, authzEnforcer)
	system.NewFileHandler(fileSvc, ecReg).RegisterRoutes(protected, authzEnforcer)

	// C 端信封示例（可选）
	if cryptoMW := setupCryptoMiddleware(aesKey, hmacKey, ecReg, rdb, cfg.RedisPrefix); cryptoMW != nil {
		cEnd := r.Group("/api/c")
		cEnd.Use(cryptoMW.Handler())
		cEnd.POST("/echo", func(c *gin.Context) {
			body, _ := c.GetRawData()
			c.JSON(200, gin.H{"echo": string(body)})
		})
	}

	return &demoApp{
		handler:      r,
		db:           db,
		rdb:          rdb,
		configCenter: configCenter,
		subCancel:    subCancel,
	}, nil
}

// ---------------------------------------------------------------------------
// 配置
// ---------------------------------------------------------------------------

type demoConfig struct {
	TablePrefix      string
	RedisPrefix      string
	Port             int
	SegmentBase      int
	AESKeyB64        string
	HMACKeyB64       string
	GCMKeyB64        string
	JWTIssuer        string
	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     int64
	JWTRefreshTTL    int64
	Argon2Params     auth.Argon2idParams
	SuperAdminRoles  []string
	HashidSalt       string
	FileRootDir      string
	FileURLPrefix    string
	FileMaxSize      int64
	FileAllowedExts  []string
	// SeedPasswords 种子用户初始密码，键=用户名。零明文默认值，缺失即 fail-fast（见 seed.go）。
	SeedPasswords map[string]string
	MySQLDSN      string
	RedisAddr     string
	RedisDB       int
	RedisPassword string
}

func loadConfig() demoConfig {
	v := viper.New()
	v.SetConfigName("config.local")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./examples/demo")
	v.SetEnvPrefix("DEMO")
	// 嵌套键 a.b.c → 环境变量 DEMO_A_B_C（使「来自环境变量」对所有键真正生效）
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("read config: %v (copy config.example.yaml to config.local.yaml)", err)
	}

	return demoConfig{
		TablePrefix:      v.GetString("app.table_prefix"),
		RedisPrefix:      v.GetString("app.redis_key_prefix"),
		Port:             v.GetInt("app.port"),
		SegmentBase:      v.GetInt("errcode.security_segment_base"),
		AESKeyB64:        v.GetString("crypto.aes_key_b64"),
		HMACKeyB64:       v.GetString("crypto.hmac_key_b64"),
		GCMKeyB64:        v.GetString("gcm.key_b64"),
		JWTIssuer:        v.GetString("jwt.issuer"),
		JWTAccessSecret:  v.GetString("jwt.access_secret"),
		JWTRefreshSecret: v.GetString("jwt.refresh_secret"),
		JWTAccessTTL:     v.GetInt64("jwt.access_ttl_seconds"),
		JWTRefreshTTL:    v.GetInt64("jwt.refresh_ttl_seconds"),
		Argon2Params: auth.Argon2idParams{
			MemoryKiB:   uint32(v.GetInt("auth.argon2id.memory_kib")),
			Iterations:  uint32(v.GetInt("auth.argon2id.iterations")),
			Parallelism: uint8(v.GetInt("auth.argon2id.parallelism")),
			SaltLen:     uint32(v.GetInt("auth.argon2id.salt_len")),
			KeyLen:      uint32(v.GetInt("auth.argon2id.key_len")),
		},
		SuperAdminRoles: v.GetStringSlice("rbac.super_admin_roles"),
		HashidSalt:      v.GetString("hashid.salt"),
		FileRootDir:     v.GetString("file.local.root_dir"),
		FileURLPrefix:   v.GetString("file.local.url_prefix"),
		FileMaxSize:     v.GetInt64("file.max_size_bytes"),
		FileAllowedExts: v.GetStringSlice("file.allowed_exts"),
		// 每个种子用户一项标量键，env 可逐项覆盖（DEMO_SEED_EDITOR_PASSWORD 等），同超管机制。
		SeedPasswords: map[string]string{
			"admin":    v.GetString("seed.admin_password"),
			"editor":   v.GetString("seed.editor_password"),
			"dept_mgr": v.GetString("seed.dept_mgr_password"),
			"biz_user": v.GetString("seed.biz_user_password"),
		},
		MySQLDSN:      v.GetString("mysql.dsn"),
		RedisAddr:     v.GetString("redis.addr"),
		RedisDB:       v.GetInt("redis.db"),
		RedisPassword: v.GetString("redis.password"),
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

// decodeB64 解码 base64 密钥；空值或长度不符返回 error（startup fail-fast）。
func decodeB64(val string, expectLen int, name string) ([]byte, error) {
	if val == "" {
		return nil, fmt.Errorf("config %s must not be empty", name)
	}
	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return nil, fmt.Errorf("config %s invalid base64: %w", name, err)
	}
	if expectLen > 0 && len(decoded) != expectLen {
		return nil, fmt.Errorf("config %s must decode to %d bytes, got %d", name, expectLen, len(decoded))
	}
	return decoded, nil
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
