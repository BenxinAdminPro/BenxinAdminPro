// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   字典 CRUD 服务 — 类型 + 项
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:15:00
// | @updated   2026-06-08 03:40:00
// | @updated   2026-06-09 17:00:00  T-004e：唯一键冲突(1062)兜底转友好码（dict_type/config_key 防 500）
// | @updated   2026-06-14 10:40:00  T-005b-4：dict/config 列表补关键字模糊 + 排序白名单；dict_data 真分页
// +----------------------------------------------------------------------

package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/benxin_dev/benxinadminpro-server/dberr"
	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"gorm.io/gorm"
)

type DictService struct {
	db   *gorm.DB
	errs *errcode.Registry
}

func NewDictService(db *gorm.DB, errs *errcode.Registry) *DictService {
	return &DictService{db: db, errs: errs}
}

// --- 类型 ---

type CreateDictTypeInput struct {
	DictType string `json:"dict_type" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Status   int8   `json:"status"`
	Remark   string `json:"remark"`
}

func (s *DictService) CreateType(ctx context.Context, in CreateDictTypeInput) (*SysDictType, error) {
	var count int64
	s.db.WithContext(ctx).Model(&SysDictType{}).Where("dict_type = ?", in.DictType).Count(&count)
	if count > 0 {
		return nil, s.errs.ErrDictTypeExists
	}
	dt := SysDictType{DictType: in.DictType, Name: in.Name, Status: in.Status, Remark: in.Remark}
	if err := s.db.WithContext(ctx).Create(&dt).Error; err != nil {
		if dberr.IsDuplicate(err) {
			return nil, s.errs.ErrDictTypeExists
		}
		return nil, fmt.Errorf("system: create dict type: %w", err)
	}
	return &dt, nil
}

// DictTypeListQuery 字典类型列表查询参数。
type DictTypeListQuery struct {
	Keyword  string // 关键字：dict_type / name 模糊
	DictType string // 字典类型编码模糊（前端列筛选 dict_type）
	Name     string // 名称模糊
	Status   *int8  // 状态精确
	Sort     string // 排序字段（白名单）
	Order    string // asc / desc
	Page     int
	PageSize int
}

func (q *DictTypeListQuery) normalize() {
	if q.Page <= 0 { q.Page = 1 }
	if q.PageSize <= 0 || q.PageSize > 100 { q.PageSize = 20 }
}

// dictTypeSortable 排序白名单：对外字段 → 真实列。
var dictTypeSortable = map[string]string{"created_at": "created_at", "id": "id"}

func (s *DictService) ListTypes(ctx context.Context, q DictTypeListQuery) ([]SysDictType, int64, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysDictType{})
	if q.Keyword != "" {
		kw := likePattern(q.Keyword)
		query = query.Where("dict_type LIKE ? OR name LIKE ?", kw, kw)
	}
	if q.DictType != "" {
		query = query.Where("dict_type LIKE ?", likePattern(q.DictType))
	}
	if q.Name != "" {
		query = query.Where("name LIKE ?", likePattern(q.Name))
	}
	if q.Status != nil {
		query = query.Where("status = ?", *q.Status)
	}
	var total int64
	query.Count(&total)
	var list []SysDictType
	query = applySort(query, q.Sort, q.Order, dictTypeSortable, "id ASC")
	query.Offset((q.Page-1)*q.PageSize).Limit(q.PageSize).Find(&list)
	return list, total, nil
}

func (s *DictService) UpdateType(ctx context.Context, id uint64, in CreateDictTypeInput) error {
	result := s.db.WithContext(ctx).Model(&SysDictType{}).Where("id = ?", id).Updates(map[string]any{
		"dict_type": in.DictType, "name": in.Name, "status": in.Status, "remark": in.Remark,
	})
	if dberr.IsDuplicate(result.Error) {
		return s.errs.ErrDictTypeExists
	}
	if result.RowsAffected == 0 {
		return s.errs.ErrDictTypeNotFound
	}
	return result.Error
}

func (s *DictService) DeleteType(ctx context.Context, id uint64) error {
	result := s.db.WithContext(ctx).Delete(&SysDictType{}, id)
	if result.RowsAffected == 0 {
		return s.errs.ErrDictTypeNotFound
	}
	return result.Error
}

// --- 项 ---

type CreateDictDataInput struct {
	DictType string `json:"dict_type" binding:"required"`
	Label    string `json:"label" binding:"required"`
	Value    string `json:"value" binding:"required"`
	Sort     int    `json:"sort"`
	Status   int8   `json:"status"`
}

func (s *DictService) CreateData(ctx context.Context, in CreateDictDataInput) (*SysDictData, error) {
	dd := SysDictData{DictType: in.DictType, Label: in.Label, Value: in.Value, Sort: in.Sort, Status: in.Status}
	if err := s.db.WithContext(ctx).Create(&dd).Error; err != nil {
		return nil, fmt.Errorf("system: create dict data: %w", err)
	}
	return &dd, nil
}

// DictDataQuery 字典项列表查询参数。
type DictDataQuery struct {
	DictType string // 必填：按字典类型筛选（前端始终携带选中类型）
	Keyword  string // 关键字：label / value 模糊
	Sort     string
	Order    string
	Page     int
	PageSize int
}

func (q *DictDataQuery) normalize() {
	if q.Page <= 0 { q.Page = 1 }
	if q.PageSize <= 0 || q.PageSize > 100 { q.PageSize = 20 }
}

// dictDataSortable 排序白名单。默认按 sort,id 升序（保留原 ListDataByType 口径）。
var dictDataSortable = map[string]string{"created_at": "created_at", "sort": "sort", "id": "id"}

// ListData 字典项真分页列表（T-005b-4：原 ListDataByType 返全量数组 → 服务端分页）。
// DictType 为空返回空集（前端未选类型时不应发请求；空 dict_type 不匹配任何项）。
func (s *DictService) ListData(ctx context.Context, q DictDataQuery) ([]SysDictData, int64, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysDictData{}).Where("dict_type = ?", q.DictType)
	if q.Keyword != "" {
		kw := likePattern(q.Keyword)
		query = query.Where("label LIKE ? OR value LIKE ?", kw, kw)
	}
	var total int64
	query.Count(&total)
	var list []SysDictData
	query = applySort(query, q.Sort, q.Order, dictDataSortable, "sort ASC, id ASC")
	query.Offset((q.Page-1)*q.PageSize).Limit(q.PageSize).Find(&list)
	return list, total, nil
}

func (s *DictService) UpdateData(ctx context.Context, id uint64, in CreateDictDataInput) error {
	return s.db.WithContext(ctx).Model(&SysDictData{}).Where("id = ?", id).Updates(map[string]any{
		"dict_type": in.DictType, "label": in.Label, "value": in.Value, "sort": in.Sort, "status": in.Status,
	}).Error
}

func (s *DictService) DeleteData(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&SysDictData{}, id).Error
}

// --- 参数 ---

type ConfigService struct {
	db   *gorm.DB
	errs *errcode.Registry
}

func NewConfigService(db *gorm.DB, errs *errcode.Registry) *ConfigService {
	return &ConfigService{db: db, errs: errs}
}

type CreateConfigInput struct {
	ConfigKey   string `json:"config_key" binding:"required"`
	ConfigValue string `json:"config_value"`
	Name        string `json:"name"`
	Remark      string `json:"remark"`
}

func (s *ConfigService) Create(ctx context.Context, in CreateConfigInput) (*SysConfig, error) {
	var count int64
	s.db.WithContext(ctx).Model(&SysConfig{}).Where("config_key = ?", in.ConfigKey).Count(&count)
	if count > 0 {
		return nil, s.errs.ErrConfigKeyExists
	}
	cfg := SysConfig{ConfigKey: in.ConfigKey, ConfigValue: in.ConfigValue, Name: in.Name, Remark: in.Remark}
	if err := s.db.WithContext(ctx).Create(&cfg).Error; err != nil {
		if dberr.IsDuplicate(err) {
			return nil, s.errs.ErrConfigKeyExists
		}
		return nil, fmt.Errorf("system: create config: %w", err)
	}
	return &cfg, nil
}

// MaskedValue 加密参数在列表/详情中的脱敏占位。
const MaskedValue = "******"

func (s *ConfigService) GetByKey(ctx context.Context, key string) (*SysConfig, error) {
	var cfg SysConfig
	err := s.db.WithContext(ctx).Where("config_key = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, s.errs.ErrConfigNotFound
	}
	if err != nil {
		return nil, err
	}
	// 加密参数脱敏：列表/详情不返回明文
	maskEncrypted(&cfg)
	return &cfg, nil
}

// ConfigListQuery 参数列表查询参数。
type ConfigListQuery struct {
	Keyword     string // 关键字：config_key / name 模糊
	ConfigKey   string // 参数键模糊（前端列筛选 config_key）
	Name        string // 参数名模糊
	IsEncrypted *int8  // 加密标志精确过滤
	Sort        string
	Order       string
	Page        int
	PageSize    int
}

func (q *ConfigListQuery) normalize() {
	if q.Page <= 0 { q.Page = 1 }
	if q.PageSize <= 0 || q.PageSize > 100 { q.PageSize = 20 }
}

var configSortable = map[string]string{"created_at": "created_at", "id": "id"}

func (s *ConfigService) List(ctx context.Context, q ConfigListQuery) ([]SysConfig, int64, error) {
	q.normalize()
	query := s.db.WithContext(ctx).Model(&SysConfig{})
	if q.Keyword != "" {
		kw := likePattern(q.Keyword)
		query = query.Where("config_key LIKE ? OR name LIKE ?", kw, kw)
	}
	if q.ConfigKey != "" {
		query = query.Where("config_key LIKE ?", likePattern(q.ConfigKey))
	}
	if q.Name != "" {
		query = query.Where("name LIKE ?", likePattern(q.Name))
	}
	if q.IsEncrypted != nil {
		query = query.Where("is_encrypted = ?", *q.IsEncrypted)
	}
	var total int64
	query.Count(&total)
	var list []SysConfig
	query = applySort(query, q.Sort, q.Order, configSortable, "id ASC")
	query.Offset((q.Page-1)*q.PageSize).Limit(q.PageSize).Find(&list)
	// 加密参数脱敏
	for i := range list {
		maskEncrypted(&list[i])
	}
	return list, total, nil
}

// maskEncrypted 对 is_encrypted=1 的参数值脱敏为 ******。
func maskEncrypted(cfg *SysConfig) {
	if cfg.IsEncrypted == 1 {
		cfg.ConfigValue = MaskedValue
	}
}

func (s *ConfigService) Update(ctx context.Context, id uint64, in CreateConfigInput) error {
	result := s.db.WithContext(ctx).Model(&SysConfig{}).Where("id = ?", id).Updates(map[string]any{
		"config_key": in.ConfigKey, "config_value": in.ConfigValue, "name": in.Name, "remark": in.Remark,
	})
	if dberr.IsDuplicate(result.Error) {
		return s.errs.ErrConfigKeyExists
	}
	if result.RowsAffected == 0 {
		return s.errs.ErrConfigNotFound
	}
	return result.Error
}

func (s *ConfigService) Delete(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&SysConfig{}, id).Error
}
