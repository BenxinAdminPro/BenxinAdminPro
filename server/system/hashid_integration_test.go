// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-004d system 对外 ID hashid 化集成测试 — 真 MySQL：出参 hashid + path :id 收口 + 防枚举
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 16:09:17
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./system/... -v -count=1 -run TestSystemHashid
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package system

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/idcodec"
	"github.com/gin-gonic/gin"
)

func sysHasher(t *testing.T) *idcodec.Hasher {
	t.Helper()
	h, _ := idcodec.NewHasher(idcodec.HashidConfig{Salt: "t-004d-int", MinLength: 8})
	return h
}

// 文件 :id 是本片安全收益核心：hashid 化叠加 RequirePerm 构成纵深防御，防枚举遍历他人文件。
// 下载用 c.Stream（需 CloseNotifier），故走真 httptest.Server 而非 ResponseRecorder。
func TestSystemHashidFileE2E_MySQL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _, _ := setupFileIntegration(t)
	h := sysHasher(t)
	reg, _ := errcode.NewRegistry(11000)
	fh := NewFileHandler(svc, reg, h)

	r := gin.New()
	r.GET("/sys/files", fh.List)
	r.GET("/sys/files/:id/download", fh.Download)
	ts := httptest.NewServer(r)
	defer ts.Close()

	content := []byte("hashid-file-content-中文")
	file, err := svc.Upload(context.Background(), "f.txt", "text/plain", int64(len(content)), bytes.NewReader(content), "admin")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	hid := h.Encode(file.ID)

	// 1) 列表出参 id 为 hashid（非裸数字）
	{
		resp, _ := http.Get(ts.URL + "/sys/files")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var parsed struct {
			Data struct {
				List []map[string]any `json:"list"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("unmarshal list: %v (%s)", err, body)
		}
		if len(parsed.Data.List) == 0 {
			t.Fatal("列表为空")
		}
		if idVal, ok := parsed.Data.List[0]["id"].(string); !ok || idVal != hid {
			t.Errorf("列表 id 应为 hashid %q，got %#v", hid, parsed.Data.List[0]["id"])
		}
	}

	// 2) 用 hashid :id 下载 → 200 且字节一致
	{
		resp, _ := http.Get(ts.URL + "/sys/files/" + hid + "/download")
		got, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("hashid 下载应 200，got %d", resp.StatusCode)
		}
		if !bytes.Equal(got, content) {
			t.Errorf("下载字节不一致")
		}
	}

	// 3) 裸数字 :id / 乱码 :id → 400（防枚举：不暴露内部 id 是否存在）
	for _, bad := range []string{"1", "999999", "!!!garbage!!!"} {
		resp, _ := http.Get(ts.URL + "/sys/files/" + bad + "/download")
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("非 hashid :id %q 应 400，got %d", bad, resp.StatusCode)
		}
	}
}

// dict 全链路：创建 → 列表出参 id hashid → 用 hashid PUT/DELETE 通；裸/乱码 :id → 400。
func TestSystemHashidDictE2E_MySQL(t *testing.T) {
	db := setupIntMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	h := sysHasher(t)
	dictSvc := NewDictService(db, reg)
	handler := NewHandler(dictSvc, NewConfigService(db, reg), NewLogService(db), reg, h)

	dt, err := dictSvc.CreateType(context.Background(), CreateDictTypeInput{DictType: "sys_yn", Name: "是否"})
	if err != nil {
		t.Fatalf("CreateType: %v", err)
	}
	hid := h.Encode(dt.ID)

	// 列表出参 id hashid
	{
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/sys/dict/types?page=1&page_size=20", nil)
		handler.ListDictTypes(c)
		if !strings.Contains(w.Body.String(), `"id":"`+hid+`"`) {
			t.Errorf("列表应含 hashid id %q，body=%s", hid, w.Body.String())
		}
		if strings.Contains(w.Body.String(), `"id":`+itoaSys(dt.ID)) {
			t.Errorf("列表不应含裸数字 id，body=%s", w.Body.String())
		}
	}

	// 用 hashid PUT → 200，落库生效
	{
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: hid}}
		c.Request = httptest.NewRequest("PUT", "/sys/dict/types/"+hid, strings.NewReader(`{"dict_type":"sys_yn","name":"是否-改"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		handler.UpdateDictType(c)
		if w.Code != http.StatusOK {
			t.Fatalf("hashid PUT 应 200，got %d (%s)", w.Code, w.Body.String())
		}
		var got SysDictType
		db.First(&got, dt.ID)
		if got.Name != "是否-改" {
			t.Errorf("PUT 应落库改名，got %q", got.Name)
		}
	}

	// 裸数字 / 乱码 :id → 400
	for _, bad := range []string{itoaSys(dt.ID), "abc123"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: bad}}
		c.Request = httptest.NewRequest("DELETE", "/sys/dict/types/"+bad, nil)
		handler.DeleteDictType(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("非 hashid :id %q 应 400，got %d", bad, w.Code)
		}
	}

	// 用 hashid DELETE → 200
	{
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: hid}}
		c.Request = httptest.NewRequest("DELETE", "/sys/dict/types/"+hid, nil)
		handler.DeleteDictType(c)
		if w.Code != http.StatusOK {
			t.Fatalf("hashid DELETE 应 200，got %d", w.Code)
		}
	}
}

func itoaSys(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
