// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   系统管理 handler — 字典/参数/操作日志/登录日志
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:30:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 13:00:00  T-003d：删本地"假"requirePerm，RegisterRoutes 注入 PermGuard 真 enforce
// +----------------------------------------------------------------------

package system

import (
	"strconv"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/gin-gonic/gin"
)

// PermGuard 鉴权中间件工厂：按 perm code 返回路由级 enforce 中间件。
// 由消费方注入（*rbac.AuthzEnforcer 结构性满足本接口）；system 不直接依赖 rbac，保持解耦。
type PermGuard interface {
	RequirePerm(code string) gin.HandlerFunc
}

// Handler 系统管理 handler。
type Handler struct {
	dictSvc   *DictService
	configSvc *ConfigService
	logSvc    *LogService
	errs      *errcode.Registry
}

// NewHandler 创建系统管理 handler。
func NewHandler(dictSvc *DictService, configSvc *ConfigService, logSvc *LogService, errs *errcode.Registry) *Handler {
	return &Handler{dictSvc: dictSvc, configSvc: configSvc, logSvc: logSvc, errs: errs}
}

// RegisterRoutes 注册系统管理路由（每条经注入的 guard 真 enforce）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, guard PermGuard) {
	// 字典
	rg.GET("/sys/dict/types", guard.RequirePerm("sys:dict:list"), h.ListDictTypes)
	rg.POST("/sys/dict/types", guard.RequirePerm("sys:dict:create"), h.CreateDictType)
	rg.PUT("/sys/dict/types/:id", guard.RequirePerm("sys:dict:update"), h.UpdateDictType)
	rg.DELETE("/sys/dict/types/:id", guard.RequirePerm("sys:dict:delete"), h.DeleteDictType)
	rg.GET("/sys/dict/data", guard.RequirePerm("sys:dict:list"), h.ListDictData)
	rg.POST("/sys/dict/data", guard.RequirePerm("sys:dict:create"), h.CreateDictData)
	rg.PUT("/sys/dict/data/:id", guard.RequirePerm("sys:dict:update"), h.UpdateDictData)
	rg.DELETE("/sys/dict/data/:id", guard.RequirePerm("sys:dict:delete"), h.DeleteDictData)

	// 参数
	rg.GET("/sys/configs", guard.RequirePerm("sys:config:list"), h.ListConfigs)
	rg.POST("/sys/configs", guard.RequirePerm("sys:config:create"), h.CreateConfig)
	rg.PUT("/sys/configs/:id", guard.RequirePerm("sys:config:update"), h.UpdateConfig)
	rg.DELETE("/sys/configs/:id", guard.RequirePerm("sys:config:delete"), h.DeleteConfig)

	// 日志
	rg.GET("/sys/logs/oper", guard.RequirePerm("sys:operlog:list"), h.ListOperLogs)
	rg.DELETE("/sys/logs/oper", guard.RequirePerm("sys:operlog:clean"), h.CleanOperLogs)
	rg.GET("/sys/logs/login", guard.RequirePerm("sys:loginlog:list"), h.ListLoginLogs)
	rg.DELETE("/sys/logs/login", guard.RequirePerm("sys:loginlog:clean"), h.CleanLoginLogs)
}

// --- 字典类型 ---

func (h *Handler) ListDictTypes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, _ := h.dictSvc.ListTypes(c.Request.Context(), page, ps)
	response.OK(c, gin.H{"list": list, "total": total, "page": page, "page_size": ps})
}

func (h *Handler) CreateDictType(c *gin.Context) {
	var in CreateDictTypeInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	dt, err := h.dictSvc.CreateType(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, dt)
}

func (h *Handler) UpdateDictType(c *gin.Context) {
	id, err := pid(c)
	if err != nil { return }
	var in CreateDictTypeInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.dictSvc.UpdateType(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *Handler) DeleteDictType(c *gin.Context) {
	id, err := pid(c)
	if err != nil { return }
	if err := h.dictSvc.DeleteType(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

// --- 字典项 ---

func (h *Handler) ListDictData(c *gin.Context) {
	dictType := c.Query("dict_type")
	list, _ := h.dictSvc.ListDataByType(c.Request.Context(), dictType)
	response.OK(c, list)
}

func (h *Handler) CreateDictData(c *gin.Context) {
	var in CreateDictDataInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	dd, err := h.dictSvc.CreateData(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, dd)
}

func (h *Handler) UpdateDictData(c *gin.Context) {
	id, err := pid(c)
	if err != nil { return }
	var in CreateDictDataInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.dictSvc.UpdateData(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *Handler) DeleteDictData(c *gin.Context) {
	id, err := pid(c)
	if err != nil { return }
	if err := h.dictSvc.DeleteData(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

// --- 参数 ---

func (h *Handler) ListConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, _ := h.configSvc.List(c.Request.Context(), page, ps)
	response.OK(c, gin.H{"list": list, "total": total, "page": page, "page_size": ps})
}

func (h *Handler) CreateConfig(c *gin.Context) {
	var in CreateConfigInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	cfg, err := h.configSvc.Create(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, cfg)
}

func (h *Handler) UpdateConfig(c *gin.Context) {
	id, err := pid(c)
	if err != nil { return }
	var in CreateConfigInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.configSvc.Update(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *Handler) DeleteConfig(c *gin.Context) {
	id, err := pid(c)
	if err != nil { return }
	if err := h.configSvc.Delete(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

// --- 日志 ---

func (h *Handler) ListOperLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	operator := c.Query("operator")
	list, total, _ := h.logSvc.ListOperLogs(c.Request.Context(), operator, nil, nil, page, ps)
	response.OK(c, gin.H{"list": list, "total": total, "page": page, "page_size": ps})
}

func (h *Handler) CleanOperLogs(c *gin.Context) {
	before := time.Now().AddDate(0, -3, 0) // 默认清 3 个月前
	rows, _ := h.logSvc.CleanOperLogs(c.Request.Context(), before)
	response.OK(c, gin.H{"deleted": rows})
}

func (h *Handler) ListLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	username := c.Query("username")
	list, total, _ := h.logSvc.ListLoginLogs(c.Request.Context(), username, page, ps)
	response.OK(c, gin.H{"list": list, "total": total, "page": page, "page_size": ps})
}

func (h *Handler) CleanLoginLogs(c *gin.Context) {
	before := time.Now().AddDate(0, -3, 0)
	rows, _ := h.logSvc.CleanLoginLogs(c.Request.Context(), before)
	response.OK(c, gin.H{"deleted": rows})
}

// --- 辅助 ---

func pid(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { response.BadReq(c) }
	return id, err
}
