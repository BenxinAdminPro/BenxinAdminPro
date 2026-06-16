// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   文件服务 — 上传/下载/列表/删除 + 上传安全校验
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:20:00
// | @updated   2026-06-14 10:48:00  T-005b-4：列表补文件名/上传人用户名模糊 + 排序 + uploader 可读化
// | @updated   2026-06-16 11:55:00  T-011b：mime 大类筛（token→后端常量前缀）+ 批量软删 BatchDelete
// +----------------------------------------------------------------------

package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/drivers/storage"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileService 文件管理服务。
type FileService struct {
	db      *gorm.DB
	driver  storage.StorageDriver
	upload  storage.UploadConfig
	errs    *errcode.Registry
	drvType string // 驱动类型标识，如 "local"
}

// NewFileService 创建文件服务。
func NewFileService(db *gorm.DB, driver storage.StorageDriver, upload storage.UploadConfig, errs *errcode.Registry, drvType string) *FileService {
	if drvType == "" {
		drvType = "local"
	}
	return &FileService{db: db, driver: driver, upload: upload, errs: errs, drvType: drvType}
}

// Upload 上传文件。
func (s *FileService) Upload(ctx context.Context, filename, contentType string, size int64, r io.Reader, uploader string) (*SysFile, error) {
	// 1. 文件名消毒
	safeName, err := storage.SanitizeFilename(filename)
	if err != nil {
		return nil, s.errs.ErrFileNameInvalid
	}

	// 2. 大小校验
	if err := s.upload.ValidateSize(size); err != nil {
		return nil, s.errs.ErrFileTooLarge
	}

	// 3. 扩展名白名单
	ext := storage.ExtFromFilename(safeName)
	if err := s.upload.ValidateExt(ext); err != nil {
		return nil, s.errs.ErrFileExtNotAllowed
	}

	// 4. Content-Type 一致性
	if err := storage.ValidateContentType(contentType, ext); err != nil {
		return nil, s.errs.ErrFileTypeMismatch
	}

	// 5. 生成 key（日期分目录 + uuid）
	now := time.Now()
	uid, _ := uuid.NewV7()
	key := fmt.Sprintf("%d/%02d/%02d/%s.%s", now.Year(), now.Month(), now.Day(), uid.String(), ext)

	// 6. 流式写入驱动（大小限制通过 LimitReader）
	limitedReader := io.LimitReader(r, s.upload.MaxSizeBytes+1) // +1 检测超限
	if err := s.driver.Put(ctx, key, limitedReader, size, contentType); err != nil {
		return nil, s.errs.ErrStorageFailed
	}

	// 7. 落元信息
	file := SysFile{
		OriginalName: safeName,
		StorageKey:   key,
		StorageType:  s.drvType,
		Size:         size,
		Mime:         contentType,
		Ext:          ext,
		Uploader:     uploader,
	}
	if err := s.db.WithContext(ctx).Create(&file).Error; err != nil {
		// 回滚：删除已写入的文件
		s.driver.Delete(ctx, key)
		return nil, fmt.Errorf("system: save file meta: %w", err)
	}

	return &file, nil
}

// Download 获取文件流。
func (s *FileService) Download(ctx context.Context, id uint64) (*SysFile, io.ReadCloser, error) {
	var file SysFile
	err := s.db.WithContext(ctx).First(&file, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, s.errs.ErrFileNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.driver.Get(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, s.errs.ErrStorageFailed
	}
	return &file, reader, nil
}

// FileListQuery 文件列表查询参数。
type FileListQuery struct {
	UploaderName string // 按上传人用户名模糊（uploader 存内部 ID，先映射 username→ID 集）
	OriginalName string // 文件名模糊
	MimeCategory string // mime 大类 token（image/video/audio/other）；未命中/空 → 不筛
	Sort         string
	Order        string
	Page         int
	PageSize     int
}

func (q *FileListQuery) normalize() {
	if q.Page <= 0 { q.Page = 1 }
	if q.PageSize <= 0 || q.PageSize > 100 { q.PageSize = 20 }
}

var fileSortable = map[string]string{"created_at": "created_at", "size": "size", "id": "id"}

// mimeCategoryPrefix 大类 token → mime 前缀（值为后端常量，绝不取用户输入拼 SQL 片段）。
var mimeCategoryPrefix = map[string]string{
	"image": "image/",
	"video": "video/",
	"audio": "audio/",
}

// applyMimeCategory 按 mime 大类 token 加筛（裁决④：token→后端常量前缀，用户输入永不进 SQL 片段）。
// image/video/audio → mime LIKE '<前缀>%'；other → 排除三大类；未命中/空 → 不加筛（同 sort 白名单回退范式）。
// 前缀为常量字面量、无 LIKE 特殊字符，故 image/video/audio 走参数化绑定、other 走纯常量 SQL，均无注入面。
func applyMimeCategory(query *gorm.DB, category string) *gorm.DB {
	switch category {
	case "image", "video", "audio":
		return query.Where("mime LIKE ?", mimeCategoryPrefix[category]+"%")
	case "other":
		return query.Where("mime NOT LIKE 'image/%' AND mime NOT LIKE 'video/%' AND mime NOT LIKE 'audio/%'")
	default:
		return query // 含空串/未知 token：不加筛，绝不报错（防探测、与 sort 回退同口径）
	}
}

// maxBatchDeleteIDs 批量软删单次 id 上限（对齐列表分页封顶，防超大请求）。
const maxBatchDeleteIDs = 100

// List 文件分页列表（T-005b-4：补文件名/上传人用户名模糊 + 排序 + uploader 可读化）。
func (s *FileService) List(ctx context.Context, q FileListQuery) ([]SysFile, int64, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysFile{})
	if q.UploaderName != "" {
		ids := userIDsByName(ctx, s.db, q.UploaderName)
		if len(ids) == 0 {
			query = query.Where("1 = 0") // 无任何用户名匹配 → 空结果
		} else {
			query = query.Where("uploader IN ?", ids)
		}
	}
	if q.OriginalName != "" {
		query = query.Where("original_name LIKE ?", likePattern(q.OriginalName))
	}
	query = applyMimeCategory(query, q.MimeCategory)
	var total int64
	query.Count(&total)
	var list []SysFile
	query = applySort(query, q.Sort, q.Order, fileSortable, "id DESC")
	query.Offset((q.Page-1)*q.PageSize).Limit(q.PageSize).Find(&list)
	s.fillUploaderNames(ctx, list)
	return list, total, nil
}

// fillUploaderNames 批量解析本页 uploader 内部 ID → 用户名展示（一次查询）。
func (s *FileService) fillUploaderNames(ctx context.Context, list []SysFile) {
	if len(list) == 0 {
		return
	}
	ids := make([]string, 0, len(list))
	for i := range list {
		if list[i].Uploader != "" {
			ids = append(ids, list[i].Uploader)
		}
	}
	names := resolveUserNames(ctx, s.db, ids)
	for i := range list {
		list[i].UploaderName = displayUserName(list[i].Uploader, names)
	}
}

// Delete 软删除文件 + 标记待物理清理。
func (s *FileService) Delete(ctx context.Context, id uint64) error {
	var file SysFile
	err := s.db.WithContext(ctx).First(&file, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.errs.ErrFileNotFound
	}
	if err != nil {
		return err
	}

	// 软删 + 标记待清理
	s.db.WithContext(ctx).Model(&SysFile{}).Where("id = ?", id).Update("status", 1)
	s.db.WithContext(ctx).Delete(&SysFile{}, id)

	// 异步物理清理
	go func() {
		if err := s.driver.Delete(context.Background(), file.StorageKey); err != nil {
			slog.Error("file_physical_delete_failed",
				slog.String("key", file.StorageKey),
				slog.String("error", err.Error()))
		}
	}()

	return nil
}

// BatchDelete 批量软删 + 逐个物理异步清理（单条 IN bulk 幂等）。
// 先取实际存在(未软删)行的 id+storage_key（模型查询带软删 scope → 已删/不存在 id 自然不入命中集 = 幂等），
// 再对命中集 status=1 + 模型 Delete（单 UPDATE deleted_at，软删 scope 自动生效；勿 .Table() 避开静默失效陷阱），
// 物理清理沿用单条 Delete 的 per-file fire-and-forget 范式（失败仅 slog、不回滚、无法纳事务）。
// 返回实际软删行数（= 命中集大小 = deleted_count）。调用方负责 id 解码/空/封顶校验。
func (s *FileService) BatchDelete(ctx context.Context, ids []uint64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	// 取真实存在且未软删的行（模型查询自动加 deleted_at IS NULL scope → 幂等基础）
	var existing []SysFile
	if err := s.db.WithContext(ctx).Model(&SysFile{}).
		Select("id", "storage_key").
		Where("id IN ?", ids).
		Find(&existing).Error; err != nil {
		return 0, err
	}
	if len(existing) == 0 {
		return 0, nil
	}
	existIDs := make([]uint64, 0, len(existing))
	for i := range existing {
		existIDs = append(existIDs, existing[i].ID)
	}

	// 标待清理 + 软删（模型 Delete，软删 scope 生效）
	s.db.WithContext(ctx).Model(&SysFile{}).Where("id IN ?", existIDs).Update("status", 1)
	res := s.db.WithContext(ctx).Where("id IN ?", existIDs).Delete(&SysFile{})
	if res.Error != nil {
		return 0, res.Error
	}

	// 逐个异步物理清理（与单条 Delete 一致）
	for i := range existing {
		key := existing[i].StorageKey
		go func(k string) {
			if err := s.driver.Delete(context.Background(), k); err != nil {
				slog.Error("file_physical_delete_failed",
					slog.String("key", k),
					slog.String("error", err.Error()))
			}
		}(key)
	}

	return int(res.RowsAffected), nil
}
