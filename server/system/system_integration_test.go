// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   T-004a 系统管理 MySQL 集成测试 — 日志落库与查询
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:05:00
// +----------------------------------------------------------------------
//
// 运行方式：go test -tags=integration ./system/... -v -count=1
// 前置：docker compose -f deploy/docker-compose.dev.yml up -d

//go:build integration

package system

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	intTestDSN    = "root:root@tcp(localhost:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local"
	intTestPrefix = "t004a_"
)

func intMigrationDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "spec", "migrations")
}

func setupIntMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(intTestDSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: intTestPrefix, SingularTable: true},
	})
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}

	tables := []string{"sys_oper_log", "sys_login_log", "sys_dict_type", "sys_dict_data", "sys_config"}
	for _, tbl := range tables {
		db.Exec("DROP TABLE IF EXISTS `" + intTestPrefix + tbl + "`")
	}

	dir := intMigrationDir()
	sqlFiles := []string{
		"T004a_sys_dict_type.sql", "T004a_sys_dict_data.sql", "T004a_sys_config.sql",
		"T004a_sys_oper_log.sql", "T004a_sys_login_log.sql",
	}
	for _, f := range sqlFiles {
		sqlBytes, _ := os.ReadFile(filepath.Join(dir, f))
		ddl := strings.ReplaceAll(string(sqlBytes), "{{TABLE_PREFIX}}", intTestPrefix)
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}

	t.Cleanup(func() {
		for _, tbl := range tables {
			db.Exec("DROP TABLE IF EXISTS `" + intTestPrefix + tbl + "`")
		}
	})
	return db
}

func TestIntegration_OperLogFallthrough(t *testing.T) {
	db := setupIntMySQL(t)
	sink := &GormOperLogSink{DB: db}
	ctx := context.Background()

	sink.Write(ctx, SysOperLog{
		Operator: "admin", Method: "POST", Path: "/sys/users",
		IP: "127.0.0.1", ReqSummary: `{"username":"test"}`, ResultCode: 0, LatencyMs: 42,
	})
	sink.Write(ctx, SysOperLog{
		Operator: "editor", Method: "PUT", Path: "/sys/posts/1",
		IP: "10.0.0.1", ResultCode: 200, LatencyMs: 15,
	})

	svc := NewLogService(db)

	// 全部
	list, total, _ := svc.ListOperLogs(ctx, "", nil, nil, 1, 10)
	if total != 2 {
		t.Errorf("total oper logs: got %d, want 2", total)
	}
	if len(list) != 2 {
		t.Errorf("list len: got %d", len(list))
	}

	// 按 operator 过滤
	list, total, _ = svc.ListOperLogs(ctx, "admin", nil, nil, 1, 10)
	if total != 1 {
		t.Errorf("filtered total: got %d, want 1", total)
	}
}

func TestIntegration_LoginLogFallthrough(t *testing.T) {
	db := setupIntMySQL(t)
	logger := NewGormLoginLogger(db)
	ctx := context.Background()

	logger.Log(ctx, auth.LoginEvent{Username: "alice", IP: "1.2.3.4", Success: true})
	logger.Log(ctx, auth.LoginEvent{Username: "bob", IP: "5.6.7.8", Success: false, Reason: "bad_credentials"})

	svc := NewLogService(db)

	list, total, _ := svc.ListLoginLogs(ctx, "", 1, 10)
	if total != 2 {
		t.Errorf("total login logs: got %d, want 2", total)
	}

	// 按 username 过滤
	list, total, _ = svc.ListLoginLogs(ctx, "alice", 1, 10)
	if total != 1 || len(list) != 1 {
		t.Errorf("filtered: total=%d len=%d", total, len(list))
	}
	if list[0].Success != 1 {
		t.Error("alice should be success=1")
	}
}
