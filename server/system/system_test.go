// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   系统管理单元测试 — 字典/参数/日志/脱敏/Registry
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:35:00
// +----------------------------------------------------------------------

package system

import (
	"context"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	db.AutoMigrate(&SysDictType{}, &SysDictData{}, &SysConfig{}, &SysOperLog{}, &SysLoginLog{})
	return db
}

func reg(t *testing.T) *errcode.Registry {
	t.Helper()
	r, _ := errcode.NewRegistry(11000)
	return r
}

// ---------------------------------------------------------------------------
// 字典
// ---------------------------------------------------------------------------

func TestDictTypeCRUD(t *testing.T) {
	db := setupDB(t)
	svc := NewDictService(db, reg(t))
	ctx := context.Background()

	dt, err := svc.CreateType(ctx, CreateDictTypeInput{DictType: "sys_status", Name: "状态"})
	if err != nil { t.Fatal(err) }
	if dt.DictType != "sys_status" { t.Error("wrong type") }

	// 重复
	_, err = svc.CreateType(ctx, CreateDictTypeInput{DictType: "sys_status", Name: "x"})
	if err == nil { t.Error("expected duplicate error") }

	// 列表
	list, total, _ := svc.ListTypes(ctx, 1, 10)
	if total != 1 || len(list) != 1 { t.Error("list wrong") }

	// 删除
	svc.DeleteType(ctx, dt.ID)
	_, total, _ = svc.ListTypes(ctx, 1, 10)
	if total != 0 { t.Error("should be deleted") }
}

func TestDictData(t *testing.T) {
	db := setupDB(t)
	svc := NewDictService(db, reg(t))
	ctx := context.Background()

	svc.CreateType(ctx, CreateDictTypeInput{DictType: "gender", Name: "性别"})
	svc.CreateData(ctx, CreateDictDataInput{DictType: "gender", Label: "男", Value: "1"})
	svc.CreateData(ctx, CreateDictDataInput{DictType: "gender", Label: "女", Value: "2"})

	list, _ := svc.ListDataByType(ctx, "gender")
	if len(list) != 2 { t.Errorf("expected 2 items, got %d", len(list)) }
}

// ---------------------------------------------------------------------------
// 参数
// ---------------------------------------------------------------------------

func TestConfigCRUD(t *testing.T) {
	db := setupDB(t)
	svc := NewConfigService(db, reg(t))
	ctx := context.Background()

	cfg, err := svc.Create(ctx, CreateConfigInput{ConfigKey: "site.name", ConfigValue: "Test", Name: "站点名"})
	if err != nil { t.Fatal(err) }
	if cfg.ConfigKey != "site.name" { t.Error("wrong key") }

	// 重复 key
	_, err = svc.Create(ctx, CreateConfigInput{ConfigKey: "site.name", ConfigValue: "x"})
	if err == nil { t.Error("expected duplicate key error") }

	// 按 key 查
	got, err := svc.GetByKey(ctx, "site.name")
	if err != nil { t.Fatal(err) }
	if got.ConfigValue != "Test" { t.Error("wrong value") }
}

// ---------------------------------------------------------------------------
// 脱敏
// ---------------------------------------------------------------------------

func TestSanitize(t *testing.T) {
	body := `{"username":"alice","password":"secret123","captcha_code":"abcd"}`
	result := Sanitize(body)

	if strings.Contains(result, "secret123") {
		t.Error("password should be sanitized")
	}
	if strings.Contains(result, "abcd") {
		t.Error("captcha_code should be sanitized")
	}
	if !strings.Contains(result, "alice") {
		t.Error("username should be preserved")
	}
	if !strings.Contains(result, "***") {
		t.Error("sanitized values should be replaced with ***")
	}
}

func TestSanitize_Token(t *testing.T) {
	body := `{"refresh_token":"eyJhbGciOiJIUzI1NiJ9.xxx"}`
	result := Sanitize(body)
	if strings.Contains(result, "eyJhbGciOiJIUzI1NiJ9") {
		t.Error("token should be sanitized")
	}
}

// ---------------------------------------------------------------------------
// 操作日志 sink
// ---------------------------------------------------------------------------

func TestGormOperLogSink(t *testing.T) {
	db := setupDB(t)
	sink := &GormOperLogSink{DB: db}

	err := sink.Write(context.Background(), SysOperLog{
		Operator: "user1", Method: "POST", Path: "/api/test",
	})
	if err != nil { t.Fatal(err) }

	var count int64
	db.Model(&SysOperLog{}).Count(&count)
	if count != 1 { t.Errorf("expected 1 log, got %d", count) }
}

// ---------------------------------------------------------------------------
// 登录日志
// ---------------------------------------------------------------------------

func TestGormLoginLogger(t *testing.T) {
	db := setupDB(t)
	logger := NewGormLoginLogger(db)
	ctx := context.Background()

	// auth 包不需导入 — 这里直接用 system 包的类型测试接口实现
	err := logger.Log(ctx, auth.LoginEvent{Username: "alice", Success: true})
	if err != nil { t.Fatal(err) }
	err = logger.Log(ctx, auth.LoginEvent{Username: "bob", Success: false, Reason: "bad_credentials"})
	if err != nil { t.Fatal(err) }

	var count int64
	db.Model(&SysLoginLog{}).Count(&count)
	if count != 2 { t.Errorf("expected 2 login logs, got %d", count) }
}

// ---------------------------------------------------------------------------
// 日志查询
// ---------------------------------------------------------------------------

func TestLogServiceList(t *testing.T) {
	db := setupDB(t)
	sink := &GormOperLogSink{DB: db}
	sink.Write(context.Background(), SysOperLog{Operator: "admin", Method: "POST", Path: "/sys/users"})
	sink.Write(context.Background(), SysOperLog{Operator: "editor", Method: "PUT", Path: "/sys/posts/1"})

	svc := NewLogService(db)
	list, total, _ := svc.ListOperLogs(context.Background(), "admin", nil, nil, 1, 10)
	if total != 1 { t.Errorf("filtered total: %d", total) }
	if len(list) != 1 { t.Errorf("filtered list: %d", len(list)) }
}
