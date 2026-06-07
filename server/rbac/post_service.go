// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   岗位 CRUD 服务 — 增删改查 + 分页 + 启用禁用
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:18:00
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/gorm"
)

// CreatePostInput 创建岗位请求。
type CreatePostInput struct {
	Code   string `json:"code" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Sort   int    `json:"sort"`
	Status int8   `json:"status"`
}

// UpdatePostInput 更新岗位请求。
type UpdatePostInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
	Status int8   `json:"status"`
}

// PostListQuery 岗位列表查询参数。
type PostListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (q *PostListQuery) normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

// PostService 岗位 CRUD 服务。
type PostService struct {
	db   *gorm.DB
	errs *errcode.Registry
}

// NewPostService 创建岗位服务。
func NewPostService(db *gorm.DB, errs *errcode.Registry) *PostService {
	return &PostService{db: db, errs: errs}
}

// Create 创建岗位。
func (s *PostService) Create(ctx context.Context, in CreatePostInput) (*SysPost, error) {
	var count int64
	s.db.WithContext(ctx).Model(&SysPost{}).Where("code = ?", in.Code).Count(&count)
	if count > 0 {
		return nil, s.errs.ErrPostCodeExists
	}

	post := SysPost{
		Code:   in.Code,
		Name:   in.Name,
		Sort:   in.Sort,
		Status: in.Status,
	}
	if err := s.db.WithContext(ctx).Create(&post).Error; err != nil {
		return nil, fmt.Errorf("rbac: create post: %w", err)
	}
	return &post, nil
}

// Get 获取岗位详情。
func (s *PostService) Get(ctx context.Context, id uint64) (*SysPost, error) {
	var post SysPost
	err := s.db.WithContext(ctx).First(&post, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, s.errs.ErrPostCodeExists // 复用：岗位不存在
	}
	if err != nil {
		return nil, fmt.Errorf("rbac: get post: %w", err)
	}
	return &post, nil
}

// List 岗位分页列表。
func (s *PostService) List(ctx context.Context, q PostListQuery) (*PageResult, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysPost{})

	var total int64
	query.Count(&total)

	var posts []SysPost
	err := query.
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Order("sort ASC, id ASC").
		Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("rbac: list posts: %w", err)
	}
	return &PageResult{List: posts, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// Update 更新岗位。
func (s *PostService) Update(ctx context.Context, id uint64, in UpdatePostInput) error {
	// 检查 code 唯一性（排除自身）
	if in.Code != "" {
		var count int64
		s.db.WithContext(ctx).Model(&SysPost{}).Where("code = ? AND id != ?", in.Code, id).Count(&count)
		if count > 0 {
			return s.errs.ErrPostCodeExists
		}
	}

	result := s.db.WithContext(ctx).Model(&SysPost{}).Where("id = ?", id).Updates(map[string]any{
		"code":   in.Code,
		"name":   in.Name,
		"sort":   in.Sort,
		"status": in.Status,
	})
	if result.RowsAffected == 0 {
		return s.errs.ErrPostCodeExists // 岗位不存在
	}
	return result.Error
}

// Delete 删除岗位。
func (s *PostService) Delete(ctx context.Context, id uint64) error {
	result := s.db.WithContext(ctx).Delete(&SysPost{}, id)
	if result.RowsAffected == 0 {
		return s.errs.ErrPostCodeExists
	}
	return result.Error
}
