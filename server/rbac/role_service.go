// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   角色 CRUD 服务 + 分配菜单/权限
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:10:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/gorm"
)

type CreateRoleInput struct {
	Code   string `json:"code" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Sort   int    `json:"sort"`
	Status int8   `json:"status"`
	Remark string `json:"remark"`
}

type UpdateRoleInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
	Status int8   `json:"status"`
	Remark string `json:"remark"`
}

type RoleListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (q *RoleListQuery) normalize() {
	if q.Page <= 0 { q.Page = 1 }
	if q.PageSize <= 0 || q.PageSize > 100 { q.PageSize = 20 }
}

type RoleService struct {
	db         *gorm.DB
	errs       *errcode.Registry
	policySync PolicySync
}

func NewRoleService(db *gorm.DB, errs *errcode.Registry, ps PolicySync) *RoleService {
	return &RoleService{db: db, errs: errs, policySync: ps}
}

func (s *RoleService) Create(ctx context.Context, in CreateRoleInput) (*SysRole, error) {
	var count int64
	s.db.WithContext(ctx).Model(&SysRole{}).Where("code = ?", in.Code).Count(&count)
	if count > 0 {
		return nil, s.errs.ErrRoleCodeExists
	}
	role := SysRole{Code: in.Code, Name: in.Name, Sort: in.Sort, Status: in.Status, Remark: in.Remark}
	if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
		return nil, fmt.Errorf("rbac: create role: %w", err)
	}
	return &role, nil
}

func (s *RoleService) Get(ctx context.Context, id uint64) (*SysRole, error) {
	var role SysRole
	err := s.db.WithContext(ctx).First(&role, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, s.errs.ErrRoleCodeExists
	}
	return &role, err
}

func (s *RoleService) List(ctx context.Context, q RoleListQuery) (*PageResult, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysRole{})
	var total int64
	query.Count(&total)
	var roles []SysRole
	err := query.Offset((q.Page-1)*q.PageSize).Limit(q.PageSize).Order("sort ASC, id ASC").Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return &PageResult{List: roles, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (s *RoleService) Update(ctx context.Context, id uint64, in UpdateRoleInput) error {
	if in.Code != "" {
		var count int64
		s.db.WithContext(ctx).Model(&SysRole{}).Where("code = ? AND id != ?", in.Code, id).Count(&count)
		if count > 0 {
			return s.errs.ErrRoleCodeExists
		}
	}
	result := s.db.WithContext(ctx).Model(&SysRole{}).Where("id = ?", id).Updates(map[string]any{
		"code": in.Code, "name": in.Name, "sort": in.Sort, "status": in.Status, "remark": in.Remark,
	})
	if result.RowsAffected == 0 {
		return s.errs.ErrRoleCodeExists
	}
	return result.Error
}

func (s *RoleService) Delete(ctx context.Context, id uint64) error {
	// 检查是否仍绑用户
	var userCount int64
	s.db.WithContext(ctx).Model(&SysUserRole{}).Where("role_id = ?", id).Count(&userCount)
	if userCount > 0 {
		return s.errs.ErrRoleInUse
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("role_id = ?", id).Delete(&SysRoleMenu{})
		return tx.Delete(&SysRole{}, id).Error
	})
}

// AssignMenus 给角色分配菜单/权限，并联动 Casbin policy。
func (s *RoleService) AssignMenus(ctx context.Context, roleID uint64, menuIDs []uint64) error {
	var role SysRole
	if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return s.errs.ErrRoleCodeExists
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("role_id = ?", roleID).Delete(&SysRoleMenu{})
		if len(menuIDs) > 0 {
			var rms []SysRoleMenu
			for _, mid := range menuIDs {
				rms = append(rms, SysRoleMenu{RoleID: roleID, MenuID: mid})
			}
			if err := tx.Create(&rms).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 联动 Casbin：从 role_menu + menu 表取 perm_code 列表
	if s.policySync != nil {
		permCodes := s.getRolePermCodes(ctx, roleID)
		return s.policySync.SyncRolePerms(ctx, role.Code, permCodes)
	}
	return nil
}

func (s *RoleService) getRolePermCodes(ctx context.Context, roleID uint64) []string {
	var codes []string
	s.db.WithContext(ctx).
		Model(&SysMenu{}).
		Joins("JOIN "+s.menuRoleTable()+" ON "+s.menuRoleTable()+".menu_id = "+s.menuTable()+".id").
		Where(s.menuRoleTable()+".role_id = ? AND "+s.menuTable()+".perm_code != '' AND "+s.menuTable()+".deleted_at IS NULL", roleID).
		Pluck(s.menuTable()+".perm_code", &codes)
	return codes
}

func (s *RoleService) menuTable() string {
	stmt := &gorm.Statement{DB: s.db}
	stmt.Parse(&SysMenu{})
	return stmt.Schema.Table
}

func (s *RoleService) menuRoleTable() string {
	stmt := &gorm.Statement{DB: s.db}
	stmt.Parse(&SysRoleMenu{})
	return stmt.Schema.Table
}
