// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   用户 CRUD 服务 — 增删改查 + 密码管理 + 启用禁用
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 19:12:00
// | @updated   2026-06-07 23:44:00
// | @updated   2026-06-09 10:40:13  T-003e：对外 ID 字段降级 json:"-"（由 handler 解码注入，非裸绑定）
// | @updated   2026-06-09 17:00:00  T-004e：唯一键冲突(1062)兜底转 ErrUsernameExists（软删/竞态防 500）
// | @updated   2026-06-14 15:30:00  T-008b：Get 预载已授角色（分配角色弹窗回填全量 role_ids 来源）
// | @updated   2026-06-14 16:40:00  T-008b 增量：List 批量回填 Roles（列表角色列，固定 2 查询非 N+1）
// | @updated   2026-06-15 17:43:54  T-009b：SetStatus 同值修复（RowsAffected==0 探测存在性区分"无变更"vs"不存在"，不再误返 404）
// +----------------------------------------------------------------------

package rbac

import (
	"context"
	"errors"
	"fmt"

	"github.com/benxin_dev/benxinadminpro-server/auth"
	"github.com/benxin_dev/benxinadminpro-server/dberr"
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
	DeptID   *uint64  `json:"-"` // 内部 ID，由 handler 解码 hashid 注入
	Status   int8     `json:"status"`
	Remark   string   `json:"remark"`
	PostIDs  []uint64 `json:"-"` // 内部 ID，由 handler 解码 hashid 注入
}

// UpdateUserInput 更新用户请求（不含密码）。
type UpdateUserInput struct {
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Email    string   `json:"email"`
	Mobile   string   `json:"mobile"`
	DeptID   *uint64  `json:"-"` // 内部 ID，由 handler 解码 hashid 注入
	Remark   string   `json:"remark"`
	PostIDs  []uint64 `json:"-"` // 内部 ID，由 handler 解码 hashid 注入
}

// UserListQuery 用户列表查询参数。
type UserListQuery struct {
	Username string     `form:"username"`
	Status   *int8      `form:"status"`
	DeptID   *uint64    `form:"-"` // 内部 ID，由 handler 解码 hashid 注入
	Page     int        `form:"page"`
	PageSize int        `form:"page_size"`
	Scope    *DataScope `form:"-"` // 由 handler 注入，禁止客户端传入
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
	db         *gorm.DB
	hasher     auth.PasswordHasher
	errs       *errcode.Registry
	policySync PolicySync // 可选，T-003b 联动 Casbin
}

// NewUserService 创建用户服务。
func NewUserService(db *gorm.DB, hasher auth.PasswordHasher, errs *errcode.Registry) *UserService {
	return &UserService{db: db, hasher: hasher, errs: errs}
}

// SetPolicySync 注入 Casbin 联动服务（可选，T-003b 启用后调用）。
func (s *UserService) SetPolicySync(ps PolicySync) {
	s.policySync = ps
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
			// username 唯一键冲突（软删记录占位/并发竞态，预检漏判）→ 友好码而非 500。
			// 在 user 写语句处就地判定，避免与 sys_user_post 复合主键冲突混淆。
			if dberr.IsDuplicate(err) {
				return s.errs.ErrUsernameExists
			}
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
		// 友好业务码直接透传——response.HandleError 用类型断言 err.(Coder) 提码，
		// 被 fmt.Errorf 包裹会断言失败落 500，故 coded 错误不可再包裹。
		if _, ok := err.(interface{ GetCode() int }); ok {
			return nil, err
		}
		return nil, fmt.Errorf("rbac: create user: %w", err)
	}

	user.PasswordHash = "" // 不返回哈希
	return &user, nil
}

// Get 获取用户详情（手动加载关联岗位）。
func (s *UserService) Get(ctx context.Context, id uint64) (*SysUser, error) {
	var user SysUser
	err := s.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, s.errs.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("rbac: get user: %w", err)
	}

	// 手动加载关联岗位（不依赖 GORM many2many association）
	var postIDs []uint64
	s.db.WithContext(ctx).Model(&SysUserPost{}).Where("user_id = ?", id).Pluck("post_id", &postIDs)
	if len(postIDs) > 0 {
		s.db.WithContext(ctx).Where("id IN ?", postIDs).Find(&user.Posts)
	}

	// T-008b：手动加载已授角色（详情专属，供「分配角色」弹窗回填全量 role_ids；
	// 仅 Get 预载、List 不载，避免列表 N+1 与出参污染）。SysRole 无敏感字段。
	var roleIDs []uint64
	s.db.WithContext(ctx).Model(&SysUserRole{}).Where("user_id = ?", id).Pluck("role_id", &roleIDs)
	if len(roleIDs) > 0 {
		s.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&user.Roles)
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

	// 数据权限过滤（自测样例：sys_user 表用 dept_id 和 id）
	if q.Scope != nil {
		query = query.Scopes(ApplyScope(q.Scope, ScopeFields{DeptColumn: "dept_id", UserColumn: "id"}))
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

	// T-008b 增量：批量回填本页用户的已授角色（列表「角色」列展示来源）。
	// 固定 2 次额外查询（junction IN + role IN），与本页行数 N 解耦，不退化成 N+1。
	s.fillUserRoles(ctx, users)

	return &PageResult{List: users, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// fillUserRoles 批量回填本页用户的 Roles（一对多分组）。
// 查询③ junction（SysUserRole 无软删，model 查询安全）：一次 WHERE user_id IN 取全页关联；
// 查询④ 角色（SysRole 有软删，model 查询让 deleted_at IS NULL scope 自动剔除软删角色，
//
//	故意不用原生 .Table()——那会绕过软删 scope，账本第 7 例潜伏点）。
//
// 内存按 user 分组回填。查询次数固定（与 N 无关），password_hash 不涉（仅查 role）。
func (s *UserService) fillUserRoles(ctx context.Context, users []SysUser) {
	if len(users) == 0 {
		return
	}
	userIDs := make([]uint64, 0, len(users))
	for i := range users {
		userIDs = append(userIDs, users[i].ID)
	}

	// 查询③：全页 user_role 关联
	var urs []SysUserRole
	s.db.WithContext(ctx).Model(&SysUserRole{}).Where("user_id IN ?", userIDs).Find(&urs)
	if len(urs) == 0 {
		return
	}
	userRoleIDs := make(map[uint64][]uint64, len(users)) // userID → []roleID
	roleIDSet := make(map[uint64]struct{})
	for _, ur := range urs {
		userRoleIDs[ur.UserID] = append(userRoleIDs[ur.UserID], ur.RoleID)
		roleIDSet[ur.RoleID] = struct{}{}
	}
	distinctRoleIDs := make([]uint64, 0, len(roleIDSet))
	for rid := range roleIDSet {
		distinctRoleIDs = append(distinctRoleIDs, rid)
	}

	// 查询④：去重后的角色（model 查询 → 软删角色自动剔除）
	var roles []SysRole
	s.db.WithContext(ctx).Where("id IN ?", distinctRoleIDs).Find(&roles)
	roleByID := make(map[uint64]SysRole, len(roles))
	for i := range roles {
		roleByID[roles[i].ID] = roles[i]
	}

	// 内存分组回填（软删角色不在 roleByID → 自动跳过）
	for i := range users {
		for _, rid := range userRoleIDs[users[i].ID] {
			if r, ok := roleByID[rid]; ok {
				users[i].Roles = append(users[i].Roles, r)
			}
		}
	}
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
//
// T-009b 修复：RowsAffected==0 不可直接判 404。MySQL 默认不带 CLIENT_FOUND_ROWS，
// 当 status 已是目标值（无变更）时同值 UPDATE 返回 0 改动行——与"记录不存在"同形。
// 旧实现一刀切返 ErrUserNotFound，使 API 直调 PUT /:id/status 传当前相同值误返 404
// （toggleStatus 永远翻转值故不触发，但直调会撞）。修法：RowsAffected==0 时显式
// 探测存在性——记录存在=幂等无变更返成功；记录不存在才返 NotFound。
// 安全/正确性：存在性校验不放宽——Count 走 Model(&SysUser{}) 带 GORM 软删 scope，
// 软删/不存在的 id 仍正确返 404，绝不一刀切返成功。
func (s *UserService) SetStatus(ctx context.Context, id uint64, status int8) error {
	result := s.db.WithContext(ctx).Model(&SysUser{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("rbac: set status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := s.db.WithContext(ctx).Model(&SysUser{}).Where("id = ?", id).Count(&count).Error; err != nil {
			return fmt.Errorf("rbac: set status existence check: %w", err)
		}
		if count == 0 {
			return s.errs.ErrUserNotFound
		}
		// 记录存在但值未变：幂等无变更，视为成功（不返 404）。
	}
	return nil
}

// AssignRoles 给用户分配角色，并联动 Casbin g 规则。
// 一致性策略：SavePolicy 失败则回滚 DB 事务 + ReloadAll 恢复 enforcer。
func (s *UserService) AssignRoles(ctx context.Context, userID uint64, roleIDs []uint64) error {
	// 提前计算 roleCodes
	var roleCodes []string
	if len(roleIDs) > 0 {
		s.db.WithContext(ctx).Model(&SysRole{}).Where("id IN ?", roleIDs).Pluck("code", &roleCodes)
	}

	// DB 事务写入
	tx := s.db.WithContext(ctx).Begin()
	tx.Where("user_id = ?", userID).Delete(&SysUserRole{})
	if len(roleIDs) > 0 {
		var urs []SysUserRole
		for _, rid := range roleIDs {
			urs = append(urs, SysUserRole{UserID: userID, RoleID: rid})
		}
		if err := tx.Create(&urs).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("rbac: assign roles: %w", err)
		}
	}

	// 联动 Casbin g 规则
	if s.policySync != nil {
		userSub := fmt.Sprintf("%d", userID)
		if err := s.policySync.SyncUserRoles(ctx, userSub, roleCodes); err != nil {
			tx.Rollback()
			s.policySync.ReloadAll(ctx)
			return fmt.Errorf("rbac: sync casbin failed, rolled back: %w", err)
		}
	}

	return tx.Commit().Error
}

func (s *UserService) userRoleTable() string {
	stmt := &gorm.Statement{DB: s.db}
	stmt.Parse(&SysUserRole{})
	return stmt.Schema.Table
}

func (s *UserService) roleTable() string {
	stmt := &gorm.Statement{DB: s.db}
	stmt.Parse(&SysRole{})
	return stmt.Schema.Table
}
