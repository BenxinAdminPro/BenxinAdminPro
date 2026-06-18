// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-004e system 唯一键冲突集成测试 — 真 MySQL 重复得友好码而非 500
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 17:00:00
// | @updated   2026-06-18 13:50:56  T-017：dict_type 禁改后移除 TestDupDictTypeUpdateRename_MySQL（改名撞键路径已不可达，同 config 先例）
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./system/... -v -count=1 -run Dup
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d
//
// dict_type/config 的 Update 无重名预检 → 改名撞已存在 key 直接触发 1062 → backstop。
// （此前该路径 RowsAffected==0 误返 NotFound 404，非 500；本片转成正确的 409 Exists。）
// 复用 intTestDSN 与 intMigrationDir()（system_integration_test.go，同包）。

//go:build integration

package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const dupSysPrefix = "t004e_"

func setupDupSysMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(intTestDSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: dupSysPrefix, SingularTable: true},
	})
	if err != nil {
		t.Fatalf("connect mysql (is docker compose running?): %v", err)
	}
	tables := []string{"sys_dict_type", "sys_config"}
	for _, tbl := range tables {
		db.Exec("DROP TABLE IF EXISTS `" + dupSysPrefix + tbl + "`")
	}
	dir := intMigrationDir()
	// 先建表，再对 sys_config 应用 is_encrypted 增列（与生产迁移序一致）
	files := []string{"T004a_sys_dict_type.sql", "T004a_sys_config.sql", "T005_sys_config_encrypted.sql"}
	for _, f := range files {
		b, _ := os.ReadFile(filepath.Join(dir, f))
		ddl := strings.ReplaceAll(string(b), "{{TABLE_PREFIX}}", dupSysPrefix)
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}
	t.Cleanup(func() {
		for _, tbl := range tables {
			db.Exec("DROP TABLE IF EXISTS `" + dupSysPrefix + tbl + "`")
		}
	})
	return db
}

func assertDupCode(t *testing.T, err error, wantCode int, label string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 期望唯一键冲突错误，得到 nil", label)
	}
	coded, ok := err.(interface{ GetCode() int })
	if !ok {
		t.Fatalf("%s: 错误非友好业务码（会渲染 500）: %v", label, err)
	}
	if coded.GetCode() != wantCode {
		t.Errorf("%s: code=%d, want=%d", label, coded.GetCode(), wantCode)
	}
	// 中立性：干净的 errcode.Error.Error() 恒为 "errcode:N"，绝不含索引名/表名/原始 SQL。
	// 用精确等值证明"未裹 DB 错误"——子串黑名单会被合法码值误伤（如 11062 含 "1062"）。
	want := fmt.Sprintf("errcode:%d", wantCode)
	if msg := err.Error(); msg != want {
		t.Errorf("%s: 错误串应为干净 %q，实得 %q（疑似裹了 DB 细节/原始 SQL）", label, want, msg)
	}
}

// 注：原 TestDupDictTypeUpdateRename_MySQL（dict_type Update 改名撞键 → ErrDictTypeExists）
// 已随 T-017 移除——UpdateDictTypeInput 不再含 DictType（dict_type 唯一键禁改，底座无级联、
// 改名会孤儿化既有 dict_data），改名场景按设计已不可能，该 backstop 路径对 dict_type 不再可达。
// 与上方 config 改名锁定（T-005b-3）同范式。Create 侧友好码仍由 TestDupDictTypeSimpleCreate_MySQL 覆盖。

// dict_type 简单重名（走预检）也应友好码。
func TestDupDictTypeSimpleCreate_MySQL(t *testing.T) {
	db := setupDupSysMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewDictService(db, reg)
	ctx := context.Background()

	if _, err := svc.CreateType(ctx, CreateDictTypeInput{DictType: "twin", Name: "A"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err := svc.CreateType(ctx, CreateDictTypeInput{DictType: "twin", Name: "B"})
	assertDupCode(t, err, reg.ErrDictTypeExists.GetCode(), "dict_type 简单重名")
}

// 注：原 TestDupConfigUpdateRename_MySQL（config Update 改名撞键 → ErrConfigKeyExists）
// 已随 T-005b-3 移除——UpdateConfigInput 不再含 config_key（编辑态键锁定，禁改，对齐前端
// disabled），改名场景按设计已不可能，该 backstop 路径对 config 不再可达。Create 侧友好码
// 仍由 TestDupConfigSimpleCreate_MySQL 覆盖。

// config 简单重名（走预检）也应友好码。
func TestDupConfigSimpleCreate_MySQL(t *testing.T) {
	db := setupDupSysMySQL(t)
	reg, _ := errcode.NewRegistry(11000)
	svc := NewConfigService(db, reg)
	ctx := context.Background()

	if _, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "twin.key", ConfigValue: "x"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "twin.key", ConfigValue: "y"})
	assertDupCode(t, err, reg.ErrConfigKeyExists.GetCode(), "config 简单重名")
}
