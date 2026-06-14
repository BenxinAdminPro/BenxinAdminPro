// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   角色 CRUD 服务 + 分配菜单/权限
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 21:10:00
// | @updated   2026-06-07 23:40:00
// | @updated   2026-06-09 17:00:00  T-004e：唯一键冲突(1062)兜底转 ErrRoleCodeExists（软删/竞态防 500）
// | @updated   2026-06-14 18:10:00  T-008c：Get 搭车预载全量 menu_ids（授权树弹窗回填来源）
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/benxin_dev/benxinadminpro-server/dberr"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/gorm"
)

type CreateRoleInput struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Sort      int    `json:"sort"`
	Status    int8   `json:"status"`
	DataScope int8   `json:"data_scope"` // 1=全部 2=本人 3=本部门，0 默认为 2
	Remark    string `json:"remark"`
}

type UpdateRoleInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Sort      int    `json:"sort"`
	Status    int8   `json:"status"`
	DataScope *int8  `json:"data_scope"` // 指针，区分未传与零值
	Remark    string `json:"remark"`
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
	// data_scope 校验：0 默认为 2（本人），非法值拒绝
	ds := in.DataScope
	if ds == 0 {
		ds = 2
	}
	if !ValidScopeType(ds) {
		return nil, s.errs.ErrInvalidDataScope
	}

	role := SysRole{Code: in.Code, Name: in.Name, Sort: in.Sort, Status: in.Status, DataScope: ds, Remark: in.Remark}
	if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
		if dberr.IsDuplicate(err) {
			return nil, s.errs.ErrRoleCodeExists
		}
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
	if err != nil {
		return nil, err
	}
	// T-008c：回填全量已授 menu_ids（M/C/F 三层，不过滤类型）供「分配菜单」授权树 setCheckedKeys。
	// AssignMenus 为全量覆写——回填必须返全集，否则漏回填的节点会在覆写提交时被静默删除（§6）。
	// 仅详情 Get 预载、List 不载（omitempty 不污染列表，同 T-008b SysUser.Roles 范式）。
	s.db.WithContext(ctx).Model(&SysRoleMenu{}).Where("role_id = ?", id).Pluck("menu_id", &role.MenuIDs)
	return &role, nil
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
	updates := map[string]any{
		"code": in.Code, "name": in.Name, "sort": in.Sort, "status": in.Status, "remark": in.Remark,
	}
	if in.DataScope != nil {
		if !ValidScopeType(*in.DataScope) {
			return s.errs.ErrInvalidDataScope
		}
		updates["data_scope"] = *in.DataScope
	}
	result := s.db.WithContext(ctx).Model(&SysRole{}).Where("id = ?", id).Updates(updates)
	if dberr.IsDuplicate(result.Error) {
		return s.errs.ErrRoleCodeExists
	}
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
// 一致性策略：先计算新 permCodes，DB 授权表写入在事务中，
// SavePolicy 失败则回滚 DB 事务 + ReloadAll 恢复 enforcer 内存。
func (s *RoleService) AssignMenus(ctx context.Context, roleID uint64, menuIDs []uint64) error {
	var role SysRole
	if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		return s.errs.ErrRoleCodeExists
	}

	// 提前计算新的 permCodes（从 menu 表读，不依赖事务内写入）
	var permCodes []string
	if len(menuIDs) > 0 {
		s.db.WithContext(ctx).Model(&SysMenu{}).
			Where("id IN ? AND perm_code != ''", menuIDs).
			Pluck("perm_code", &permCodes)
	}

	// DB 事务写入授权表
	tx := s.db.WithContext(ctx).Begin()
	tx.Where("role_id = ?", roleID).Delete(&SysRoleMenu{})
	if len(menuIDs) > 0 {
		var rms []SysRoleMenu
		for _, mid := range menuIDs {
			rms = append(rms, SysRoleMenu{RoleID: roleID, MenuID: mid})
		}
		if err := tx.Create(&rms).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("rbac: assign menus: %w", err)
		}
	}

	// 联动 Casbin
	if s.policySync != nil {
		if err := s.policySync.SyncRolePerms(ctx, role.Code, permCodes); err != nil {
			tx.Rollback()
			// SavePolicy 失败，回滚 DB + 恢复 enforcer 内存
			s.policySync.ReloadAll(ctx)
			return fmt.Errorf("rbac: sync casbin failed, rolled back: %w", err)
		}
	}

	return tx.Commit().Error
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
