// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   菜单/权限点 CRUD 服务 — 树形 + menu_type 校验
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:12:00
// | @updated   2026-06-09 10:40:13  T-003e：parent_id 降级 json:"-"（由 handler 解码注入）
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
	db   *gorm.DB
	errs *errcode.Registry
}

func NewMenuService(db *gorm.DB, errs *errcode.Registry) *MenuService {
	return &MenuService{db: db, errs: errs}
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

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
	})
}

func (s *MenuService) Delete(ctx context.Context, id uint64) error {
	var childCount int64
	s.db.WithContext(ctx).Model(&SysMenu{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return s.errs.ErrMenuHasChildren
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("menu_id = ?", id).Delete(&SysRoleMenu{})
		return tx.Delete(&SysMenu{}, id).Error
	})
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
