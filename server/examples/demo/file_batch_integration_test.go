// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-011b 批量软删 + mime 大类筛 e2e — enforce(dept_mgr 200↔editor 403) + 幂等 + 负例
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-16 11:55:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./examples/demo/ -v -count=1 -run TestFileBatch
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 验证：① POST /sys/files/batch-delete enforce：dept_mgr 200 ↔ editor 403（复用 sys:file:delete）
//       ② 端到端：上传 3 文件 → 批量软删 deleted_count=3 → list 不再出现；重复删幂等 deleted_count=0
//       ③ 负例：空数组 400 / 非法 hashid 400 / 超封顶 100 → 400
//       ④ mime 大类筛 HTTP：?mime_category=image 仅回 image/*（jpg/png），不回 txt
// 隔离：独立表前缀 demofb_ + Redis DB 14。

//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/rbac"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	fbPrefix  = "demofb_"
	fbRedisDB = 14
)

func TestFileBatchE2E(t *testing.T) {
	cfg := e2eConfig(t)
	cfg.TablePrefix = fbPrefix
	cfg.RedisDB = fbRedisDB

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

	mgrTok := loginTok(t, ts.URL, "dept_mgr", e2eDeptMgrPwd)
	editorTok := loginTok(t, ts.URL, "editor", e2eEditorPwd)

	// 上传两张图 + 一个 txt（dept_mgr 持 sys:file:upload）
	idPng := uploadFileE2E(t, ts.URL, mgrTok, "p1.png", "image/png")
	idTxt := uploadFileE2E(t, ts.URL, mgrTok, "n1.txt", "text/plain")
	idJpg := uploadFileE2E(t, ts.URL, mgrTok, "p2.jpg", "image/jpeg")

	// ④ mime 大类筛：image 仅回 2 张图，不回 txt
	imgList := doJSON(t, http.MethodGet, ts.URL+"/sys/files?mime_category=image&page=1&page_size=50", mgrTok, nil)
	if imgList.status != http.StatusOK {
		t.Fatalf("list image: %d %+v", imgList.status, imgList.body)
	}
	if total, _ := dataMap(t, imgList)["total"].(float64); total != 2 {
		t.Errorf("mime_category=image total=%v, want 2", dataMap(t, imgList)["total"])
	}
	// other 仅回 txt
	otherList := doJSON(t, http.MethodGet, ts.URL+"/sys/files?mime_category=other&page=1&page_size=50", mgrTok, nil)
	if total, _ := dataMap(t, otherList)["total"].(float64); total != 1 {
		t.Errorf("mime_category=other total=%v, want 1", dataMap(t, otherList)["total"])
	}
	t.Log("mime 大类筛 OK: image=2（jpg+png）/ other=1（txt）")

	// ① enforce：editor POST batch-delete → 403（被 guard 拦在 handler 前，文件未动）
	ed := doJSON(t, http.MethodPost, ts.URL+"/sys/files/batch-delete", editorTok, map[string]any{
		"ids": []string{idPng, idTxt, idJpg},
	})
	if ed.status != http.StatusForbidden {
		t.Fatalf("editor batch-delete: got %d, want 403", ed.status)
	}
	// 文件仍在（editor 被拦未删）
	all := doJSON(t, http.MethodGet, ts.URL+"/sys/files?page=1&page_size=50", mgrTok, nil)
	if total, _ := dataMap(t, all)["total"].(float64); total != 3 {
		t.Fatalf("after editor 403, files should remain 3, got %v", dataMap(t, all)["total"])
	}

	// ② dept_mgr 批量删 3 个 → 200 deleted_count=3
	del := doJSON(t, http.MethodPost, ts.URL+"/sys/files/batch-delete", mgrTok, map[string]any{
		"ids": []string{idPng, idTxt, idJpg},
	})
	if del.status != http.StatusOK {
		t.Fatalf("dept_mgr batch-delete: got %d, want 200 (%+v)", del.status, del.body)
	}
	if dc, _ := dataMap(t, del)["deleted_count"].(float64); dc != 3 {
		t.Errorf("deleted_count=%v, want 3", dataMap(t, del)["deleted_count"])
	}
	// 软删行从列表消失
	after := doJSON(t, http.MethodGet, ts.URL+"/sys/files?page=1&page_size=50", mgrTok, nil)
	if total, _ := dataMap(t, after)["total"].(float64); total != 0 {
		t.Errorf("after batch delete total=%v, want 0", dataMap(t, after)["total"])
	}
	t.Log("enforce + e2e OK: editor 403 文件未动 ↔ dept_mgr 200 deleted_count=3 列表清空")

	// 幂等：重复删已删 id → 200 deleted_count=0
	again := doJSON(t, http.MethodPost, ts.URL+"/sys/files/batch-delete", mgrTok, map[string]any{
		"ids": []string{idPng, idTxt, idJpg},
	})
	if again.status != http.StatusOK {
		t.Fatalf("idempotent batch-delete: got %d, want 200", again.status)
	}
	if dc, _ := dataMap(t, again)["deleted_count"].(float64); dc != 0 {
		t.Errorf("idempotent deleted_count=%v, want 0", dataMap(t, again)["deleted_count"])
	}

	// ③ 负例：空数组 400 / 非法 hashid 400 / 超 100 400
	empty := doJSON(t, http.MethodPost, ts.URL+"/sys/files/batch-delete", mgrTok, map[string]any{"ids": []string{}})
	if empty.status != http.StatusBadRequest {
		t.Errorf("empty ids: got %d, want 400", empty.status)
	}
	bad := doJSON(t, http.MethodPost, ts.URL+"/sys/files/batch-delete", mgrTok, map[string]any{
		"ids": []string{"not-a-valid-hashid"},
	})
	if bad.status != http.StatusBadRequest {
		t.Errorf("invalid hashid: got %d, want 400", bad.status)
	}
	over := make([]string, 101)
	for i := range over {
		over[i] = idPng // 内容无所谓，101 个先被封顶拦下
	}
	ov := doJSON(t, http.MethodPost, ts.URL+"/sys/files/batch-delete", mgrTok, map[string]any{"ids": over})
	if ov.status != http.StatusBadRequest {
		t.Errorf("over cap(101): got %d, want 400", ov.status)
	}
	t.Log("负例 OK: 空数组 400 / 非法 hashid 400 / 超封顶 101 → 400")

	t.Log("ALL PASSED: T-011b 批量软删 enforce + 幂等 + mime 大类筛 + 负例端到端")
}

// uploadFileE2E 经 HTTP multipart 上传一个文件，返回其对外 hashid。
func uploadFileE2E(t *testing.T, baseURL, tok, filename, contentType string) string {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	hdr.Set("Content-Type", contentType)
	fw, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatalf("create form part: %v", err)
	}
	fw.Write([]byte("batch-delete-probe-content"))
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/sys/files", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload %s: %v", filename, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload %s: expected 200, got %d", filename, resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	data, _ := out["data"].(map[string]any)
	id, _ := data["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		t.Fatalf("upload %s: empty id in response (body=%s)", filename, string(raw))
	}
	return id
}
