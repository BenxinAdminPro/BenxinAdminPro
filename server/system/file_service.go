// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   文件服务 — 上传/下载/列表/删除 + 上传安全校验
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:20:00
// | @updated   2026-06-14 10:48:00  T-005b-4：列表补文件名/上传人用户名模糊 + 排序 + uploader 可读化
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
