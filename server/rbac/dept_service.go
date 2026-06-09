// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   部门 CRUD 服务 — 树形结构 + ancestors 同步 + 防环
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:15:00
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

// CreateDeptInput 创建部门请求。
type CreateDeptInput struct {
	ParentID uint64 `json:"-"` // 内部 ID，由 handler 解码 hashid 注入
	Name     string `json:"name" binding:"required"`
	Sort     int    `json:"sort"`
	Leader   string `json:"leader"`
	Status   int8   `json:"status"`
}

// UpdateDeptInput 更新部门请求。
type UpdateDeptInput struct {
	ParentID *uint64 `json:"-"` // 内部 ID（指针保留移动语义），由 handler 解码 hashid 注入
	Name     string  `json:"name"`
	Sort     int     `json:"sort"`
	Leader   string  `json:"leader"`
	Status   int8    `json:"status"`
}

// DeptService 部门 CRUD 服务。
type DeptService struct {
	db   *gorm.DB
	errs *errcode.Registry
}

// NewDeptService 创建部门服务。
func NewDeptService(db *gorm.DB, errs *errcode.Registry) *DeptService {
	return &DeptService{db: db, errs: errs}
}

// Create 创建部门。
func (s *DeptService) Create(ctx context.Context, in CreateDeptInput) (*SysDept, error) {
	ancestors := "0"
	if in.ParentID > 0 {
		parent, err := s.getByID(ctx, in.ParentID)
		if err != nil {
			return nil, s.errs.ErrInvalidParentDept
		}
		ancestors = parent.Ancestors + "," + strconv.FormatUint(parent.ID, 10)
	}

	dept := SysDept{
		ParentID:  in.ParentID,
		Ancestors: ancestors,
		Name:      in.Name,
		Sort:      in.Sort,
		Leader:    in.Leader,
		Status:    in.Status,
	}
	if err := s.db.WithContext(ctx).Create(&dept).Error; err != nil {
		return nil, fmt.Errorf("rbac: create dept: %w", err)
	}
	return &dept, nil
}

// Get 获取部门详情。
func (s *DeptService) Get(ctx context.Context, id uint64) (*SysDept, error) {
	return s.getByID(ctx, id)
}

// Tree 获取部门树。
func (s *DeptService) Tree(ctx context.Context) ([]*SysDept, error) {
	var depts []SysDept
	if err := s.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("rbac: list depts: %w", err)
	}
	return buildTree(depts), nil
}

// Update 更新部门（含移动节点）。
func (s *DeptService) Update(ctx context.Context, id uint64, in UpdateDeptInput) error {
	dept, err := s.getByID(ctx, id)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"name":   in.Name,
			"sort":   in.Sort,
			"leader": in.Leader,
			"status": in.Status,
		}

		// 移动节点
		if in.ParentID != nil && *in.ParentID != dept.ParentID {
			newParentID := *in.ParentID

			// 防环：不能把节点挂到自己的子孙下
			if newParentID > 0 {
				selfStr := strconv.FormatUint(id, 10)
				// 查新父节点
				var newParent SysDept
				if err := tx.First(&newParent, newParentID).Error; err != nil {
					return s.errs.ErrInvalidParentDept
				}
				// 新父节点的 ancestors 包含自己 → 形成环
				if strings.Contains(newParent.Ancestors+","+strconv.FormatUint(newParent.ID, 10), ","+selfStr+",") ||
					strings.HasSuffix(newParent.Ancestors+","+strconv.FormatUint(newParent.ID, 10), ","+selfStr) ||
					newParent.ID == id {
					return s.errs.ErrInvalidParentDept
				}
				updates["parent_id"] = newParentID
				newAncestors := newParent.Ancestors + "," + strconv.FormatUint(newParent.ID, 10)
				updates["ancestors"] = newAncestors

				// 同步子树 ancestors
				oldPrefix := dept.Ancestors + "," + selfStr
				newPrefix := newAncestors + "," + selfStr
				if err := tx.Model(&SysDept{}).
					Where("ancestors LIKE ?", oldPrefix+"%").
					Update("ancestors", gorm.Expr("REPLACE(ancestors, ?, ?)", oldPrefix, newPrefix)).Error; err != nil {
					return fmt.Errorf("rbac: update subtree ancestors: %w", err)
				}
			} else {
				updates["parent_id"] = 0
				updates["ancestors"] = "0"
				// 同步子树
				oldPrefix := dept.Ancestors + "," + strconv.FormatUint(id, 10)
				newPrefix := "0," + strconv.FormatUint(id, 10)
				if err := tx.Model(&SysDept{}).
					Where("ancestors LIKE ?", oldPrefix+"%").
					Update("ancestors", gorm.Expr("REPLACE(ancestors, ?, ?)", oldPrefix, newPrefix)).Error; err != nil {
					return fmt.Errorf("rbac: update subtree ancestors: %w", err)
				}
			}
		}

		return tx.Model(&SysDept{}).Where("id = ?", id).Updates(updates).Error
	})
}

// Delete 删除部门（有子节点或关联用户则拒绝）。
func (s *DeptService) Delete(ctx context.Context, id uint64) error {
	// 有子部门？
	var childCount int64
	s.db.WithContext(ctx).Model(&SysDept{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return s.errs.ErrDeptHasChildren
	}

	// 有关联用户？
	var userCount int64
	s.db.WithContext(ctx).Model(&SysUser{}).Where("dept_id = ?", id).Count(&userCount)
	if userCount > 0 {
		return s.errs.ErrDeptHasUsers
	}

	result := s.db.WithContext(ctx).Delete(&SysDept{}, id)
	if result.RowsAffected == 0 {
		return s.errs.ErrInvalidParentDept // 部门不存在
	}
	return result.Error
}

// ---------------------------------------------------------------------------
// 内部方法
// ---------------------------------------------------------------------------

func (s *DeptService) getByID(ctx context.Context, id uint64) (*SysDept, error) {
	var dept SysDept
	err := s.db.WithContext(ctx).First(&dept, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, s.errs.ErrInvalidParentDept
	}
	if err != nil {
		return nil, fmt.Errorf("rbac: get dept: %w", err)
	}
	return &dept, nil
}

// buildTree 从扁平列表构建树。
func buildTree(depts []SysDept) []*SysDept {
	nodeMap := make(map[uint64]*SysDept, len(depts))
	var roots []*SysDept

	for i := range depts {
		d := &depts[i]
		nodeMap[d.ID] = d
	}
	for _, d := range nodeMap {
		if d.ParentID == 0 {
			roots = append(roots, d)
		} else if parent, ok := nodeMap[d.ParentID]; ok {
			parent.Children = append(parent.Children, d)
		} else {
			roots = append(roots, d) // 孤儿节点当根
		}
	}
	return roots
}
