// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   dberr 单测 — 1062/ErrDuplicatedKey 命中、其它错误不误转
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 17:00:00
// +----------------------------------------------------------------------

package dberr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestIsDuplicate(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"mysql 1062 直接", &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'x' for key 'idx_sys_user_username'"}, true},
		{"mysql 1062 被 fmt.Errorf 包裹", fmt.Errorf("rbac: create user: %w", &mysql.MySQLError{Number: 1062}), true},
		{"gorm.ErrDuplicatedKey", gorm.ErrDuplicatedKey, true},
		{"gorm.ErrDuplicatedKey 被包裹", fmt.Errorf("wrap: %w", gorm.ErrDuplicatedKey), true},
		{"mysql 1452 外键不误转", &mysql.MySQLError{Number: 1452, Message: "foreign key"}, false},
		{"mysql 1364 字段无默认值不误转", &mysql.MySQLError{Number: 1364}, false},
		{"gorm.ErrRecordNotFound 不误转", gorm.ErrRecordNotFound, false},
		{"普通 error 不误转", errors.New("connection refused"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsDuplicate(c.err); got != c.want {
				t.Errorf("IsDuplicate(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
