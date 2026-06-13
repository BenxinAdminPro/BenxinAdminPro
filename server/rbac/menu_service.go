// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   菜单/权限点 CRUD 服务 — 树形 + menu_type 校验
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:12:00
// | @updated   2026-06-09 10:40:13  T-003e：parent_id 降级 json:"-"（由 handler 解码注入）
// | @updated   2026-06-13 15:30:00  T-005b-1：菜单 CUD 接 Casbin policy 重载联动（改/删 F 节点权限即时收回）
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/gorm"
)

type CreateMenuInput struct {
	ParentID  uint64 `json:"-"` // 内部 ID，由 handler 解码 hashid 注入
	MenuType  string `json:"menu_type" binding:"required"`
	Name      string `json:"name" binding:"required"`
	PermCode  string `json:"perm_code"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	Visible   int8   `json:"visible"`
	Status    int8   `json:"status"`
}

type UpdateMenuInput struct {
	ParentID  *uint64 `json:"-"` // 内部 ID（指针保留移动语义），由 handler 解码 hashid 注入
	MenuType  string  `json:"menu_type"`
	Name      string  `json:"name"`
	PermCode  string  `json:"perm_code"`
	Path      string  `json:"path"`
	Component string  `json:"component"`
	Icon      string  `json:"icon"`
	Sort      int     `json:"sort"`
	Visible   int8    `json:"visible"`
	Status    int8    `json:"status"`
}

type MenuService struct {
	db         *gorm.DB
	errs       *errcode.Registry
	policySync PolicySync // 可选，T-005b-1 菜单 CUD 后联动 Casbin policy 重载
}

func NewMenuService(db *gorm.DB, errs *errcode.Registry) *MenuService {
	return &MenuService{db: db, errs: errs}
}

// SetPolicySync 注入 Casbin 联动服务（可选，与 UserService.SetPolicySync 同范式）。
// 注入后，菜单写路径 CUD 在变更提交后触发 policy 重载，使改/删 menu_type=F 节点的
// perm_code 后 casbin p 规则即时同步——修 T-003b 既有安全缺陷（MenuService 原无 policySync，
// 删 F 节点意图收回权限但旧 perm_code 的 p 规则滞留 → 已授角色仍 enforce 通过、"以为关了门其实没关"）。
// 未注入（nil）时退化为原行为（不重载），保留既有单测无 policySync 路径绿。
func (s *MenuService) SetPolicySync(ps PolicySync) {
	s.policySync = ps
}

// syncPolicy 在菜单写操作（已提交）后全量重建 Casbin policy。
// 设计说明（T-005b-1）：
//   - 仅 menu_type=F 节点持 perm_code、进 policy（M/C 强制空）。改 F 的 perm_code 会使旧码 p 规则
//     滞留、新码失联；删 F 会使该 perm_code 的 p 规则滞留 → 权限收不回。本片接全量 ReloadAll 求正确。
//   - 全量 vs 增量：菜单 CUD 为后台配置态、极低频；ReloadAll 从 role_menu JOIN menu 真相重建，
//     天然正确（不会误删其它角色/其它 perm_code 的 policy）、与 role AssignMenus/user AssignRoles
//     共用同一 PolicySync 范式。增量（只动变化 perm_code 的 p 规则）复杂且易引新 bug，非必要不做。
//   - create 实为 no-op：新建菜单尚无 role_menu 关联 → ReloadAll 不增 p 规则（真正增量发生在后续
//     AssignMenus）。此处仍统一调用，保写路径一致、为将来"建时挂默认角色"等场景预留，不特判。
//   - 一致性（T-003b 铁律）：ReloadAll 严格在菜单变更提交后执行、读已提交真相，成功即收敛到与菜单
//     一致；不存在 casbin 领先于未提交菜单写的窗口（优于 AssignMenus 的 sync-before-commit 顺序）。
//     重载失败则返回错误（菜单已提交、policy 滞留可由下次写或重启 NewEnforcer→ReloadAll 幂等恢复），
//     与既有 AssignMenus 残留口径同级——ReloadAll 读 s.db 不能见未提交 tx，故无法做"读 tx 内未提交态"
//     的真原子回滚（需改 PolicySync 接口签名，超本片 scope）。
func (s *MenuService) syncPolicy(ctx context.Context) error {
	if s.policySync == nil {
		return nil
	}
	if err := s.policySync.ReloadAll(ctx); err != nil {
		return fmt.Errorf("rbac: menu policy reload: %w", err)
	}
	return nil
}

func (s *MenuService) Create(ctx context.Context, in CreateMenuInput) (*SysMenu, error) {
	if err := s.validateMenuType(ctx, in.MenuType, in.PermCode, 0); err != nil {
		return nil, err
	}

	ancestors := "0"
	if in.ParentID > 0 {
		parent, err := s.getByID(ctx, in.ParentID)
		if err != nil {
			return nil, s.errs.ErrInvalidParentDept
		}
		ancestors = parent.Ancestors + "," + strconv.FormatUint(parent.ID, 10)
	}

	menu := SysMenu{
		ParentID: in.ParentID, Ancestors: ancestors, MenuType: in.MenuType,
		Name: in.Name, PermCode: in.PermCode, Path: in.Path, Component: in.Component,
		Icon: in.Icon, Sort: in.Sort, Visible: in.Visible, Status: in.Status,
	}
	if err := s.db.WithContext(ctx).Create(&menu).Error; err != nil {
		return nil, fmt.Errorf("rbac: create menu: %w", err)
	}
	if err := s.syncPolicy(ctx); err != nil {
		return nil, err
	}
	return &menu, nil
}

func (s *MenuService) Tree(ctx context.Context) ([]*SysMenu, error) {
	var menus []SysMenu
	if err := s.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		return nil, err
	}
	return buildMenuTree(menus), nil
}

func (s *MenuService) Update(ctx context.Context, id uint64, in UpdateMenuInput) error {
	menu, err := s.getByID(ctx, id)
	if err != nil {
		return err
	}

	menuType := in.MenuType
	if menuType == "" {
		menuType = menu.MenuType
	}
	if err := s.validateMenuType(ctx, menuType, in.PermCode, id); err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"menu_type": menuType, "name": in.Name, "perm_code": in.PermCode,
			"path": in.Path, "component": in.Component, "icon": in.Icon,
			"sort": in.Sort, "visible": in.Visible, "status": in.Status,
		}

		if in.ParentID != nil && *in.ParentID != menu.ParentID {
			newParentID := *in.ParentID
			if newParentID > 0 {
				selfStr := strconv.FormatUint(id, 10)
				var newParent SysMenu
				if err := tx.First(&newParent, newParentID).Error; err != nil {
					return s.errs.ErrInvalidParentDept
				}
				fullPath := newParent.Ancestors + "," + strconv.FormatUint(newParent.ID, 10)
				if strings.Contains(fullPath, ","+selfStr+",") ||
					strings.HasSuffix(fullPath, ","+selfStr) || newParent.ID == id {
					return s.errs.ErrInvalidParentDept
				}
				updates["parent_id"] = newParentID
				newAncestors := fullPath
				updates["ancestors"] = newAncestors
				oldPrefix := menu.Ancestors + "," + selfStr
				newPrefix := newAncestors + "," + selfStr
				tx.Model(&SysMenu{}).Where("ancestors LIKE ?", oldPrefix+"%").
					Update("ancestors", gorm.Expr("REPLACE(ancestors, ?, ?)", oldPrefix, newPrefix))
			} else {
				updates["parent_id"] = 0
				updates["ancestors"] = "0"
			}
		}

		return tx.Model(&SysMenu{}).Where("id = ?", id).Updates(updates).Error
	}); err != nil {
		return err
	}
	// 菜单已提交后重载 policy：改 F 节点 perm_code → 旧码 p 规则消失、新码进入（修 T-003b 缺陷）
	return s.syncPolicy(ctx)
}

func (s *MenuService) Delete(ctx context.Context, id uint64) error {
	var childCount int64
	s.db.WithContext(ctx).Model(&SysMenu{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return s.errs.ErrMenuHasChildren
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("menu_id = ?", id).Delete(&SysRoleMenu{})
		return tx.Delete(&SysMenu{}, id).Error
	}); err != nil {
		return err
	}
	// 菜单+授权已提交后重载 policy：删 F 节点 → 该 perm_code 的 p 规则消失，权限真收回（修 T-003b 缺陷）
	return s.syncPolicy(ctx)
}

// GetUserMenuTree 获取用户可见菜单树（基于用户角色聚合）。
func (s *MenuService) GetUserMenuTree(ctx context.Context, userID uint64) ([]*SysMenu, error) {
	menuIDs := s.getUserMenuIDs(ctx, userID)
	if len(menuIDs) == 0 {
		return nil, nil
	}
	var menus []SysMenu
	s.db.WithContext(ctx).Where("id IN ? AND menu_type IN ? AND status = 0", menuIDs, []string{MenuTypeDir, MenuTypePage}).
		Order("sort ASC, id ASC").Find(&menus)
	return buildMenuTree(menus), nil
}

// GetUserPermCodes 获取用户权限码集合。
func (s *MenuService) GetUserPermCodes(ctx context.Context, userID uint64) ([]string, error) {
	menuIDs := s.getUserMenuIDs(ctx, userID)
	if len(menuIDs) == 0 {
		return nil, nil
	}
	var codes []string
	s.db.WithContext(ctx).Model(&SysMenu{}).
		Where("id IN ? AND perm_code != '' AND status = 0", menuIDs).
		Pluck("perm_code", &codes)
	return codes, nil
}

func (s *MenuService) getUserMenuIDs(ctx context.Context, userID uint64) []uint64 {
	// user → user_role → role → role_menu → menu
	var menuIDs []uint64
	s.db.WithContext(ctx).Model(&SysRoleMenu{}).
		Select("DISTINCT "+s.roleMenuTable()+".menu_id").
		Joins("JOIN "+s.userRoleTable()+" ON "+s.userRoleTable()+".role_id = "+s.roleMenuTable()+".role_id").
		Where(s.userRoleTable()+".user_id = ?", userID).
		Pluck(s.roleMenuTable()+".menu_id", &menuIDs)
	return menuIDs
}

func (s *MenuService) getByID(ctx context.Context, id uint64) (*SysMenu, error) {
	var menu SysMenu
	err := s.db.WithContext(ctx).First(&menu, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, s.errs.ErrInvalidParentDept
	}
	return &menu, err
}

func (s *MenuService) validateMenuType(ctx context.Context, menuType, permCode string, excludeID uint64) error {
	if menuType == MenuTypeButton {
		if permCode == "" {
			return s.errs.ErrMenuPermRequired
		}
		// perm_code 唯一性
		q := s.db.WithContext(ctx).Model(&SysMenu{}).Where("perm_code = ?", permCode)
		if excludeID > 0 {
			q = q.Where("id != ?", excludeID)
		}
		var count int64
		q.Count(&count)
		if count > 0 {
			return s.errs.ErrPermCodeExists
		}
	} else if menuType == MenuTypeDir || menuType == MenuTypePage {
		if permCode != "" {
			return s.errs.ErrMenuPermForbidden
		}
	}
	return nil
}

func (s *MenuService) roleMenuTable() string {
	stmt := &gorm.Statement{DB: s.db}
	stmt.Parse(&SysRoleMenu{})
	return stmt.Schema.Table
}

func (s *MenuService) userRoleTable() string {
	stmt := &gorm.Statement{DB: s.db}
	stmt.Parse(&SysUserRole{})
	return stmt.Schema.Table
}

func buildMenuTree(menus []SysMenu) []*SysMenu {
	nodeMap := make(map[uint64]*SysMenu, len(menus))
	var roots []*SysMenu
	for i := range menus {
		nodeMap[menus[i].ID] = &menus[i]
	}
	for _, m := range nodeMap {
		if m.ParentID == 0 {
			roots = append(roots, m)
		} else if parent, ok := nodeMap[m.ParentID]; ok {
			parent.Children = append(parent.Children, m)
		} else {
			roots = append(roots, m)
		}
	}
	return roots
}
