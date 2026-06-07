// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   用户 CRUD 服务 — 增删改查 + 密码管理 + 启用禁用
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:12:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/gorm"
)

// CreateUserInput 创建用户请求。
type CreateUserInput struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Email    string   `json:"email"`
	Mobile   string   `json:"mobile"`
	DeptID   *uint64  `json:"dept_id"`
	Status   int8     `json:"status"`
	Remark   string   `json:"remark"`
	PostIDs  []uint64 `json:"post_ids"`
}

// UpdateUserInput 更新用户请求（不含密码）。
type UpdateUserInput struct {
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Email    string   `json:"email"`
	Mobile   string   `json:"mobile"`
	DeptID   *uint64  `json:"dept_id"`
	Remark   string   `json:"remark"`
	PostIDs  []uint64 `json:"post_ids"`
}

// UserListQuery 用户列表查询参数。
type UserListQuery struct {
	Username string `form:"username"`
	Status   *int8  `form:"status"`
	DeptID   *uint64 `form:"dept_id"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

func (q *UserListQuery) normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// UserService 用户 CRUD 服务。
type UserService struct {
	db     *gorm.DB
	hasher auth.PasswordHasher
	errs   *errcode.Registry
}

// NewUserService 创建用户服务。
func NewUserService(db *gorm.DB, hasher auth.PasswordHasher, errs *errcode.Registry) *UserService {
	return &UserService{db: db, hasher: hasher, errs: errs}
}

// Create 创建用户。
func (s *UserService) Create(ctx context.Context, in CreateUserInput) (*SysUser, error) {
	// 唯一性校验
	var count int64
	s.db.WithContext(ctx).Model(&SysUser{}).Where("username = ?", in.Username).Count(&count)
	if count > 0 {
		return nil, s.errs.ErrUsernameExists
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("rbac: hash password: %w", err)
	}

	user := SysUser{
		Username:     in.Username,
		PasswordHash: hash,
		Nickname:     in.Nickname,
		Avatar:       in.Avatar,
		Email:        in.Email,
		Mobile:       in.Mobile,
		DeptID:       in.DeptID,
		Status:       in.Status,
		Remark:       in.Remark,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		// 关联岗位
		if len(in.PostIDs) > 0 {
			var userPosts []SysUserPost
			for _, pid := range in.PostIDs {
				userPosts = append(userPosts, SysUserPost{UserID: user.ID, PostID: pid})
			}
			return tx.Create(&userPosts).Error
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("rbac: create user: %w", err)
	}

	user.PasswordHash = "" // 不返回哈希
	return &user, nil
}

// Get 获取用户详情。
func (s *UserService) Get(ctx context.Context, id uint64) (*SysUser, error) {
	var user SysUser
	err := s.db.WithContext(ctx).Preload("Posts").First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, s.errs.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("rbac: get user: %w", err)
	}
	user.PasswordHash = ""
	return &user, nil
}

// List 用户分页列表。
func (s *UserService) List(ctx context.Context, q UserListQuery) (*PageResult, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysUser{})

	if q.Username != "" {
		query = query.Where("username LIKE ?", "%"+q.Username+"%")
	}
	if q.Status != nil {
		query = query.Where("status = ?", *q.Status)
	}
	if q.DeptID != nil {
		query = query.Where("dept_id = ?", *q.DeptID)
	}

	var total int64
	query.Count(&total)

	var users []SysUser
	err := query.
		Select("id, username, nickname, avatar, email, mobile, dept_id, status, remark, created_at, updated_at").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Order("id DESC").
		Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("rbac: list users: %w", err)
	}

	return &PageResult{List: users, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// Update 更新用户（不含密码）。
func (s *UserService) Update(ctx context.Context, id uint64, in UpdateUserInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&SysUser{}).Where("id = ?", id).Updates(map[string]any{
			"nickname": in.Nickname,
			"avatar":   in.Avatar,
			"email":    in.Email,
			"mobile":   in.Mobile,
			"dept_id":  in.DeptID,
			"remark":   in.Remark,
		})
		if result.Error != nil {
			return fmt.Errorf("rbac: update user: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return s.errs.ErrUserNotFound
		}

		// 更新岗位关联
		if in.PostIDs != nil {
			tx.Where("user_id = ?", id).Delete(&SysUserPost{})
			if len(in.PostIDs) > 0 {
				var userPosts []SysUserPost
				for _, pid := range in.PostIDs {
					userPosts = append(userPosts, SysUserPost{UserID: id, PostID: pid})
				}
				tx.Create(&userPosts)
			}
		}
		return nil
	})
}

// Delete 软删除用户。
func (s *UserService) Delete(ctx context.Context, id uint64) error {
	result := s.db.WithContext(ctx).Delete(&SysUser{}, id)
	if result.Error != nil {
		return fmt.Errorf("rbac: delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return s.errs.ErrUserNotFound
	}
	// 清理岗位关联
	s.db.WithContext(ctx).Where("user_id = ?", id).Delete(&SysUserPost{})
	return nil
}

// ResetPassword 重置/修改密码。
func (s *UserService) ResetPassword(ctx context.Context, id uint64, newPassword string) error {
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("rbac: hash password: %w", err)
	}
	result := s.db.WithContext(ctx).Model(&SysUser{}).Where("id = ?", id).Update("password_hash", hash)
	if result.Error != nil {
		return fmt.Errorf("rbac: reset password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return s.errs.ErrUserNotFound
	}
	return nil
}

// SetStatus 启用/禁用用户。
func (s *UserService) SetStatus(ctx context.Context, id uint64, status int8) error {
	result := s.db.WithContext(ctx).Model(&SysUser{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("rbac: set status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return s.errs.ErrUserNotFound
	}
	return nil
}
