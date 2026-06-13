// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   uploader 恒空修复 e2e 验证 — T-005b-2 auth.Claims.GetSubject() 落值真跑
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-13 15:30:00
// +----------------------------------------------------------------------
//
// 运行方式：
//   go test -tags=integration ./examples/demo/ -v -count=1 -run TestUploaderFill
//
// 验证目标（修 T-007f 暴露缺陷）：补 auth.Claims.GetSubject() 后，文件上传 handler 的鸭子类型断言
//   interface{ GetSubject() string } 成立，sys_file.uploader 真落值（= 上传者 Subject），不再恒空。
// 隔离：独立表前缀 demoup_ + Redis DB 7。

//go:build integration

package main

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	upPrefix  = "demoup_"
	upRedisDB = 7
)

func TestUploaderFillE2E(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = upPrefix
	cfg.RedisDB = upRedisDB

	ctx := context.Background()
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), rbac.NewDBConfig(cfg.TablePrefix))
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	dropPrefixedTables(t, db, cfg.TablePrefix)
	t.Cleanup(func() { dropPrefixedTables(t, db, cfg.TablePrefix) })

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	rdb.FlushDB(ctx)
	t.Cleanup(func() { rdb.FlushDB(ctx); rdb.Close() })

	app, err := buildApp(cfg, db, rdb)
	if err != nil {
		t.Fatalf("buildApp: %v", err)
	}
	t.Cleanup(app.subCancel)
	ts := httptest.NewServer(app.handler)
	t.Cleanup(ts.Close)

	// dept_mgr 持有 sys:file:upload（全量授权），且非超管走真上传链路
	login := doJSON(t, http.MethodPost, ts.URL+"/auth/login", "", map[string]string{
		"username": "dept_mgr", "password": e2eDeptMgrPwd,
	})
	if login.status != http.StatusOK {
		t.Fatalf("login dept_mgr: %d %+v", login.status, login.body)
	}
	tok, _ := dataMap(t, login)["access_token"].(string)
	if tok == "" {
		t.Fatalf("empty token")
	}
	// dept_mgr 的内部 ID（Subject 即 user.ID 字符串）
	var mgr rbac.SysUser
	if err := db.Where("username = ?", "dept_mgr").First(&mgr).Error; err != nil {
		t.Fatalf("load dept_mgr: %v", err)
	}
	mgrSubject := strconv.FormatUint(mgr.ID, 10) // Subject = user.ID 字符串形态

	// ---- 构造 multipart 上传（字段名 file，.txt 在白名单内）----
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="uploader_probe.txt"`)
	hdr.Set("Content-Type", "text/plain") // 显式声明，过 Content-Type 一致校验（默认 octet-stream 会 415）
	fw, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatalf("create form part: %v", err)
	}
	fw.Write([]byte("uploader fill probe content"))
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/sys/files", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// ---- DB 实查 sys_file.uploader：必须非空且 == 上传者 Subject(=user.ID) ----
	type fileRow struct {
		OriginalName string
		Uploader     string
	}
	var row fileRow
	if err := db.Table(cfg.TablePrefix+"sys_file").
		Select("original_name, uploader").
		Where("original_name = ?", "uploader_probe.txt").
		Scan(&row).Error; err != nil {
		t.Fatalf("query sys_file: %v", err)
	}
	if row.Uploader == "" {
		t.Fatalf("uploader 恒空缺陷未修：sys_file.uploader 为空（GetSubject() 断言仍失败）")
	}
	if row.Uploader != mgrSubject {
		t.Fatalf("uploader 落值不符：expected %q (dept_mgr Subject), got %q", mgrSubject, row.Uploader)
	}
	t.Logf("uploader OK: sys_file.uploader=%q（非空、= 上传者 Subject）", row.Uploader)

	// ---- uploader 过滤可用：GET /sys/files?uploader=<id> 应能筛到该文件 ----
	listed := doJSON(t, http.MethodGet, ts.URL+"/sys/files?uploader="+mgrSubject, tok, nil)
	if listed.status != http.StatusOK {
		t.Fatalf("list by uploader: %d %+v", listed.status, listed.body)
	}
	d := dataMap(t, listed)
	total, _ := d["total"].(float64)
	if total < 1 {
		t.Fatalf("uploader 过滤不可用：按 uploader=%s 过滤 total=%v，期望 ≥1", mgrSubject, d["total"])
	}
	t.Logf("uploader 过滤 OK: GET /sys/files?uploader=%s total=%v", mgrSubject, d["total"])

	t.Log("ALL PASSED: uploader 真落值 + 过滤可用（T-005b-2 GetSubject() 修复生效）")
}
