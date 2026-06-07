// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   字典 CRUD 服务 — 类型 + 项
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:15:00
// | @updated   2026-06-08 03:40:00
// +----------------------------------------------------------------------

package system

import (
	"context"
	"errors"
	"fmt"

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
		return nil, fmt.Errorf("system: create dict type: %w", err)
	}
	return &dt, nil
}

func (s *DictService) ListTypes(ctx context.Context, page, pageSize int) ([]SysDictType, int64, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 || pageSize > 100 { pageSize = 20 }
	var total int64
	s.db.WithContext(ctx).Model(&SysDictType{}).Count(&total)
	var list []SysDictType
	s.db.WithContext(ctx).Offset((page-1)*pageSize).Limit(pageSize).Order("id ASC").Find(&list)
	return list, total, nil
}

func (s *DictService) UpdateType(ctx context.Context, id uint64, in CreateDictTypeInput) error {
	result := s.db.WithContext(ctx).Model(&SysDictType{}).Where("id = ?", id).Updates(map[string]any{
		"dict_type": in.DictType, "name": in.Name, "status": in.Status, "remark": in.Remark,
	})
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

func (s *DictService) ListDataByType(ctx context.Context, dictType string) ([]SysDictData, error) {
	var list []SysDictData
	err := s.db.WithContext(ctx).Where("dict_type = ?", dictType).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
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

func (s *ConfigService) List(ctx context.Context, page, pageSize int) ([]SysConfig, int64, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 || pageSize > 100 { pageSize = 20 }
	var total int64
	s.db.WithContext(ctx).Model(&SysConfig{}).Count(&total)
	var list []SysConfig
	s.db.WithContext(ctx).Offset((page-1)*pageSize).Limit(pageSize).Order("id ASC").Find(&list)
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
	if result.RowsAffected == 0 {
		return s.errs.ErrConfigNotFound
	}
	return result.Error
}

func (s *ConfigService) Delete(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&SysConfig{}, id).Error
}
