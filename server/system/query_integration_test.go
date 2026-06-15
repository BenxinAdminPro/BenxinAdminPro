// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-005b-4 列表查询能力 + 操作人可读化集成测试 — 真 MySQL 端到端
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-14 11:10:00
// | @updated   2026-06-15 21:07:05  T-010a：qeDSN 改读 BENXIN_TEST_MYSQL_DSN（testsupport 收口，默认不变）
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./system/... -v -count=1 -run TestQueryEnhance
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// 覆盖：① operator/uploader 内部 ID → username 可读化（含已软删→「已注销」、空→「匿名」）
//       ② 按用户名模糊过滤（operator/uploader）③ 排序白名单 asc/desc ④ 时间范围
//       ⑤ 排序注入负例（分号/非白名单 → 回退默认，绝不进 ORDER BY）⑥ dict_data 真分页

//go:build integration

package system

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/drivers/storage"
	"github.com/benxin_dev/benxinadminpro-server/internal/testsupport"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const qePrefix = "t005b4_"

// qeDSN：优先 BENXIN_TEST_MYSQL_DSN，缺省本地默认。
var qeDSN = testsupport.MySQLDSN()

func setupQueryEnhance(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(qeDSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: qePrefix, SingularTable: true},
	})
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	tables := []string{"sys_user", "sys_oper_log", "sys_file", "sys_dict_data", "sys_dict_type"}
	drop := func() {
		for _, tbl := range tables {
			db.Exec("DROP TABLE IF EXISTS `" + qePrefix + tbl + "`")
		}
	}
	drop()
	dir := filepath.Join("..", "spec", "migrations")
	for _, f := range []string{
		"T003a_sys_user.sql", "T004a_sys_oper_log.sql", "T004b_sys_file.sql",
		"T004a_sys_dict_type.sql", "T004a_sys_dict_data.sql",
	} {
		sqlBytes, _ := os.ReadFile(filepath.Join(dir, f))
		ddl := strings.ReplaceAll(string(sqlBytes), "{{TABLE_PREFIX}}", qePrefix)
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}
	t.Cleanup(drop)
	return db
}

// 直插用户行（指定 id），含一个软删用户验证「已注销」回落。
func seedUser(t *testing.T, db *gorm.DB, id uint64, username string, deleted bool) {
	t.Helper()
	tbl := qePrefix + "sys_user"
	if deleted {
		if err := db.Exec("INSERT INTO `"+tbl+"` (id, username, deleted_at) VALUES (?, ?, NOW())", id, username).Error; err != nil {
			t.Fatalf("seed deleted user: %v", err)
		}
		return
	}
	if err := db.Exec("INSERT INTO `"+tbl+"` (id, username) VALUES (?, ?)", id, username).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// ① operator 可读化 + ② 用户名过滤 + ③ 排序 + ⑤ 注入负例（操作日志）
func TestQueryEnhance_OperLogResolveAndFilter(t *testing.T) {
	db := setupQueryEnhance(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice", false)
	seedUser(t, db, 2, "bob", false)
	seedUser(t, db, 3, "carol", true) // 软删用户 → 应回落「已注销」

	sink := &GormOperLogSink{DB: db}
	// 时间错开以验证排序
	sink.Write(ctx, SysOperLog{Operator: "1", Method: "POST", Path: "/sys/users", LatencyMs: 30})
	time.Sleep(1100 * time.Millisecond)
	sink.Write(ctx, SysOperLog{Operator: "2", Method: "PUT", Path: "/sys/posts/1", LatencyMs: 10})
	sink.Write(ctx, SysOperLog{Operator: "3", Method: "DELETE", Path: "/sys/roles/1", LatencyMs: 50})
	sink.Write(ctx, SysOperLog{Operator: "", Method: "POST", Path: "/sys/anon", LatencyMs: 5}) // 空操作人

	svc := NewLogService(db)

	// ① 可读化：operator_name 解析
	all, total, _ := svc.ListOperLogs(ctx, OperLogQuery{Page: 1, PageSize: 10})
	if total != 4 {
		t.Fatalf("total: got %d want 4", total)
	}
	got := map[string]string{} // operator(id) → operator_name
	for _, e := range all {
		got[e.Operator] = e.OperatorName
	}
	if got["1"] != "alice" {
		t.Errorf("operator 1 → name: got %q want alice", got["1"])
	}
	if got["3"] != DeletedUserName {
		t.Errorf("软删用户 3 → name: got %q want %q", got["3"], DeletedUserName)
	}
	if got[""] != AnonymousName {
		t.Errorf("空 operator → name: got %q want %q", got[""], AnonymousName)
	}

	// ② 按用户名模糊过滤（"ali" → alice 的 id=1）
	flist, ftotal, _ := svc.ListOperLogs(ctx, OperLogQuery{OperatorName: "ali", Page: 1, PageSize: 10})
	if ftotal != 1 || len(flist) != 1 || flist[0].Operator != "1" {
		t.Errorf("用户名过滤 ali：total=%d 应=1，命中 operator=%q 应=1", ftotal, firstOper(flist))
	}
	// 无匹配用户名 → 空结果（而非不过滤）
	_, ztotal, _ := svc.ListOperLogs(ctx, OperLogQuery{OperatorName: "nobody-xyz", Page: 1, PageSize: 10})
	if ztotal != 0 {
		t.Errorf("无匹配用户名应空结果，got total=%d", ztotal)
	}

	// ③ 排序：latency_ms asc → 5,10,30,50
	asc, _, _ := svc.ListOperLogs(ctx, OperLogQuery{Sort: "latency_ms", Order: "asc", Page: 1, PageSize: 10})
	if len(asc) != 4 || asc[0].LatencyMs != 5 || asc[3].LatencyMs != 50 {
		t.Errorf("latency asc 排序错: %v", latencies(asc))
	}

	// ⑤ 注入负例：sort 带分号/非白名单 → 回退默认 id DESC，绝不进 ORDER BY（不报错、不注入）
	inj, itotal, _ := svc.ListOperLogs(ctx, OperLogQuery{Sort: "id; DROP TABLE x;--", Order: "asc", Page: 1, PageSize: 10})
	if itotal != 4 || len(inj) != 4 {
		t.Fatalf("注入 sort 应被忽略仍正常返回，got total=%d", itotal)
	}
	// 默认 id DESC：最后插入的（空操作人那条）id 最大，应在首位
	if inj[0].Path != "/sys/anon" {
		t.Errorf("注入 sort 应回退默认 id DESC，首行 path=%q 应=/sys/anon", inj[0].Path)
	}
}

// ④ 时间范围（操作日志）— 用显式 created_at 直插，避开 DATETIME 秒级截断与时区歧义。
func TestQueryEnhance_OperLogTimeRange(t *testing.T) {
	db := setupQueryEnhance(t)
	ctx := context.Background()
	tbl := qePrefix + "sys_oper_log"
	db.Exec("INSERT INTO `"+tbl+"` (operator, path, created_at) VALUES (?,?,?)", "", "/old", "2020-01-01 00:00:00")
	db.Exec("INSERT INTO `"+tbl+"` (operator, path, created_at) VALUES (?,?,?)", "", "/new", "2030-01-01 00:00:00")

	cut := time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
	svc := NewLogService(db)

	// end_time=2025 → 仅 2020 那条
	older, total, _ := svc.ListOperLogs(ctx, OperLogQuery{EndTime: &cut, Page: 1, PageSize: 10})
	if total != 1 || older[0].Path != "/old" {
		t.Errorf("end<=2025 应只 /old 1 条, got total=%d", total)
	}
	// start_time=2025 → 仅 2030 那条
	newer, total2, _ := svc.ListOperLogs(ctx, OperLogQuery{StartTime: &cut, Page: 1, PageSize: 10})
	if total2 != 1 || newer[0].Path != "/new" {
		t.Errorf("start>=2025 应只 /new 1 条, got total=%d", total2)
	}
}

// ② uploader 可读化 + 用户名过滤（文件）
func TestQueryEnhance_FileUploaderResolve(t *testing.T) {
	db := setupQueryEnhance(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice", false)
	seedUser(t, db, 9, "ghost", true) // 软删

	// 直插文件元信息（绕过真磁盘上传，仅测列表解析）
	tbl := qePrefix + "sys_file"
	db.Exec("INSERT INTO `"+tbl+"` (original_name, storage_key, storage_type, size, uploader) VALUES (?,?,?,?,?)",
		"a.txt", "2026/06/14/x.txt", "local", 10, "1")
	db.Exec("INSERT INTO `"+tbl+"` (original_name, storage_key, storage_type, size, uploader) VALUES (?,?,?,?,?)",
		"b.txt", "2026/06/14/y.txt", "local", 20, "9")
	db.Exec("INSERT INTO `"+tbl+"` (original_name, storage_key, storage_type, size, uploader) VALUES (?,?,?,?,?)",
		"c.txt", "2026/06/14/z.txt", "local", 30, "")

	// List 仅查 DB，不触 driver/upload；driver 传 nil、upload 空配置即可。
	svc := NewFileService(db, nil, storage.UploadConfig{}, reg(t), "local")

	list, total, _ := svc.List(ctx, FileListQuery{Page: 1, PageSize: 10})
	if total != 3 {
		t.Fatalf("total files: got %d want 3", total)
	}
	byName := map[string]string{}
	for _, f := range list {
		byName[f.OriginalName] = f.UploaderName
	}
	if byName["a.txt"] != "alice" {
		t.Errorf("uploader 1 → %q want alice", byName["a.txt"])
	}
	if byName["b.txt"] != DeletedUserName {
		t.Errorf("软删 uploader 9 → %q want %q", byName["b.txt"], DeletedUserName)
	}
	if byName["c.txt"] != AnonymousName {
		t.Errorf("空 uploader → %q want %q", byName["c.txt"], AnonymousName)
	}

	// 按用户名过滤
	flist, ftotal, _ := svc.List(ctx, FileListQuery{UploaderName: "alice", Page: 1, PageSize: 10})
	if ftotal != 1 || len(flist) != 1 || flist[0].OriginalName != "a.txt" {
		t.Errorf("uploader 用户名过滤 alice 失败：total=%d", ftotal)
	}
}

// ⑥ dict_data 真分页
func TestQueryEnhance_DictDataPagination(t *testing.T) {
	db := setupQueryEnhance(t)
	ctx := context.Background()
	svc := NewDictService(db, reg(t))
	svc.CreateType(ctx, CreateDictTypeInput{DictType: "color", Name: "颜色"})
	for i := 0; i < 25; i++ {
		svc.CreateData(ctx, CreateDictDataInput{DictType: "color", Label: "L", Value: "v", Sort: i})
	}
	// 第 1 页 10 条 + total=25
	p1, total, _ := svc.ListData(ctx, DictDataQuery{DictType: "color", Page: 1, PageSize: 10})
	if total != 25 || len(p1) != 10 {
		t.Errorf("page1: total=%d len=%d 应 25/10", total, len(p1))
	}
	// 第 3 页应剩 5 条
	p3, _, _ := svc.ListData(ctx, DictDataQuery{DictType: "color", Page: 3, PageSize: 10})
	if len(p3) != 5 {
		t.Errorf("page3 len=%d 应 5", len(p3))
	}
	// 关键字模糊（无匹配）
	_, ktotal, _ := svc.ListData(ctx, DictDataQuery{DictType: "color", Keyword: "nomatch", Page: 1, PageSize: 10})
	if ktotal != 0 {
		t.Errorf("keyword 无匹配应 0, got %d", ktotal)
	}
}

// --- 小工具 ---

func firstOper(list []SysOperLog) string {
	if len(list) == 0 {
		return ""
	}
	return list[0].Operator
}

func latencies(list []SysOperLog) []int {
	out := make([]int, len(list))
	for i, e := range list {
		out[i] = e.LatencyMs
	}
	return out
}
