// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   Casbin enforcer 接入 — gorm-adapter + model.conf 加载
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 14:12:00
// +----------------------------------------------------------------------

package rbac

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// NewEnforcer 创建 Casbin enforcer 实例。
//   - db: GORM 数据库连接
//   - modelPath: model.conf 文件路径（通常为 server/spec/rbac/model.conf）
//   - tablePrefix: 表前缀（如 "kp_"），由配置注入，禁硬编码
//
// 适配器显式关闭自动建表（TurnOffAutoMigrate），建表只走 spec/migrations SQL。
func NewEnforcer(db *gorm.DB, modelPath string, tablePrefix string) (*casbin.Enforcer, error) {
	tableName := tablePrefix + "casbin_rule"

	// 在创建适配器前关闭自动建表，建表只走 spec/migrations SQL
	gormadapter.TurnOffAutoMigrate(db)

	adapter, err := gormadapter.NewAdapterByDBWithCustomTable(db, nil, tableName)
	if err != nil {
		return nil, fmt.Errorf("rbac: create gorm adapter: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("rbac: create enforcer: %w", err)
	}

	// 从 DB 加载策略
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("rbac: load policy: %w", err)
	}

	return enforcer, nil
}
