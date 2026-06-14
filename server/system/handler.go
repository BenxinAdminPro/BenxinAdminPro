// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   系统管理 handler — 字典/参数/操作日志/登录日志
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 00:30:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 13:00:00  T-003d：删本地"假"requirePerm，RegisterRoutes 注入 PermGuard 真 enforce
// | @updated   2026-06-09 16:09:17  T-004d：对外 ID hashid 化 — path :id 解码 + 出参 encoder（注入 idcodec hasher）
// | @updated   2026-06-14 10:52:00  T-005b-4：dict/config/日志列表补查询参数（模糊/时间范围/排序）+ dict_data 真分页
// +----------------------------------------------------------------------

package system

import (
	"strconv"
	"time"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/idcodec"
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
	hasher    *idcodec.Hasher  // 注入；对外 ID hashid 编解码（nil 退化为裸 id）
	enc       *ResponseEncoder // 始终非 nil；内部容忍 hasher==nil
}

// NewHandler 创建系统管理 handler。
// T-004d：新增 hasher 注入参数（对外契约变更）——消费方装配须注入 idcodec.NewHasher 实例；
// 传 nil 则出参/入参 id 退化为裸 uint64（与未启用 hashid 的部署对称）。
func NewHandler(dictSvc *DictService, configSvc *ConfigService, logSvc *LogService, errs *errcode.Registry, hasher *idcodec.Hasher) *Handler {
	return &Handler{
		dictSvc:   dictSvc,
		configSvc: configSvc,
		logSvc:    logSvc,
		errs:      errs,
		hasher:    hasher,
		enc:       NewResponseEncoder(hasher),
	}
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
	list, total, _ := h.dictSvc.ListTypes(c.Request.Context(), DictTypeListQuery{
		Keyword:  c.Query("keyword"),
		DictType: c.Query("dict_type"),
		Name:     c.Query("name"),
		Status:   queryInt8Ptr(c, "status"),
		Sort:     c.Query("sort"),
		Order:    c.Query("order"),
		Page:     page,
		PageSize: ps,
	})
	response.OK(c, h.enc.Page(list, total, page, ps))
}

func (h *Handler) CreateDictType(c *gin.Context) {
	var in CreateDictTypeInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	dt, err := h.dictSvc.CreateType(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, h.enc.Item(dt))
}

func (h *Handler) UpdateDictType(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil { return }
	var in CreateDictTypeInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.dictSvc.UpdateType(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *Handler) DeleteDictType(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil { return }
	if err := h.dictSvc.DeleteType(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

// --- 字典项 ---

func (h *Handler) ListDictData(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, _ := h.dictSvc.ListData(c.Request.Context(), DictDataQuery{
		DictType: c.Query("dict_type"),
		Keyword:  c.Query("keyword"),
		Sort:     c.Query("sort"),
		Order:    c.Query("order"),
		Page:     page,
		PageSize: ps,
	})
	response.OK(c, h.enc.Page(list, total, page, ps))
}

func (h *Handler) CreateDictData(c *gin.Context) {
	var in CreateDictDataInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	dd, err := h.dictSvc.CreateData(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, h.enc.Item(dd))
}

func (h *Handler) UpdateDictData(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil { return }
	var in CreateDictDataInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.dictSvc.UpdateData(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *Handler) DeleteDictData(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil { return }
	if err := h.dictSvc.DeleteData(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

// --- 参数 ---

func (h *Handler) ListConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, _ := h.configSvc.List(c.Request.Context(), ConfigListQuery{
		Keyword:     c.Query("keyword"),
		ConfigKey:   c.Query("config_key"),
		Name:        c.Query("name"),
		IsEncrypted: queryInt8Ptr(c, "is_encrypted"),
		Sort:        c.Query("sort"),
		Order:       c.Query("order"),
		Page:        page,
		PageSize:    ps,
	})
	response.OK(c, h.enc.Page(list, total, page, ps))
}

func (h *Handler) CreateConfig(c *gin.Context) {
	var in CreateConfigInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	cfg, err := h.configSvc.Create(c.Request.Context(), in)
	if err != nil { response.ErrResp(c, err); return }
	response.OK(c, h.enc.Item(cfg))
}

func (h *Handler) UpdateConfig(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil { return }
	var in CreateConfigInput
	if err := c.ShouldBindJSON(&in); err != nil { response.BadReq(c); return }
	if err := h.configSvc.Update(c.Request.Context(), id, in); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

func (h *Handler) DeleteConfig(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil { return }
	if err := h.configSvc.Delete(c.Request.Context(), id); err != nil { response.ErrResp(c, err); return }
	response.OK(c, nil)
}

// --- 日志 ---

func (h *Handler) ListOperLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	start, end, ok := parseTimeRange(c)
	if !ok {
		return // parseTimeRange 已写 400
	}
	list, total, _ := h.logSvc.ListOperLogs(c.Request.Context(), OperLogQuery{
		OperatorName: c.Query("operator_name"),
		Path:         c.Query("path"),
		StartTime:    start,
		EndTime:      end,
		Sort:         c.Query("sort"),
		Order:        c.Query("order"),
		Page:         page,
		PageSize:     ps,
	})
	response.OK(c, h.enc.Page(list, total, page, ps))
}

func (h *Handler) CleanOperLogs(c *gin.Context) {
	before := time.Now().AddDate(0, -3, 0) // 默认清 3 个月前
	rows, _ := h.logSvc.CleanOperLogs(c.Request.Context(), before)
	response.OK(c, gin.H{"deleted": rows})
}

func (h *Handler) ListLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	start, end, ok := parseTimeRange(c)
	if !ok {
		return
	}
	list, total, _ := h.logSvc.ListLoginLogs(c.Request.Context(), LoginLogQuery{
		Username:  c.Query("username"),
		IP:        c.Query("ip"),
		Success:   queryInt8Ptr(c, "success"),
		StartTime: start,
		EndTime:   end,
		Sort:      c.Query("sort"),
		Order:     c.Query("order"),
		Page:      page,
		PageSize:  ps,
	})
	response.OK(c, h.enc.Page(list, total, page, ps))
}

// --- 查询参数辅助 ---

// queryInt8Ptr 读取可选 int8 查询参数；缺省/非法返回 nil（不参与过滤）。
func queryInt8Ptr(c *gin.Context, key string) *int8 {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	n, err := strconv.ParseInt(raw, 10, 8)
	if err != nil {
		return nil
	}
	v := int8(n)
	return &v
}

// parseTimeRange 解析 start_time/end_time 查询参数并校验区间。
// 解析失败或 start>end → 写 400 并返回 ok=false（不静默吞）。end 为纯日期时补到当日末。
func parseTimeRange(c *gin.Context) (start, end *time.Time, ok bool) {
	start, err := parseTimeParam(c.Query("start_time"), false)
	if err != nil {
		response.BadReq(c)
		return nil, nil, false
	}
	end, err = parseTimeParam(c.Query("end_time"), true)
	if err != nil {
		response.BadReq(c)
		return nil, nil, false
	}
	if start != nil && end != nil && start.After(*end) {
		response.BadReq(c)
		return nil, nil, false
	}
	return start, end, true
}

func (h *Handler) CleanLoginLogs(c *gin.Context) {
	before := time.Now().AddDate(0, -3, 0)
	rows, _ := h.logSvc.CleanLoginLogs(c.Request.Context(), before)
	response.OK(c, gin.H{"deleted": rows})
}

