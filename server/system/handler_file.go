// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   文件管理 handler — 上传/下载/列表/删除
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:25:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 13:00:00  T-003d：RegisterRoutes 注入 PermGuard，路由走真 enforce
// | @updated   2026-06-09 16:09:17  T-004d：对外 ID hashid 化 — :id 解码 + 出参 encoder（注入 idcodec hasher）
// | @updated   2026-06-14 10:55:00  T-005b-4：列表补文件名/上传人用户名模糊 + 排序（FileListQuery）
// | @updated   2026-06-16 11:55:00  T-011b：列表 mime_category 大类筛 + 批量软删端点 POST /sys/files/batch-delete
// +----------------------------------------------------------------------

package system

import (
	"io"
	"net/http"
	"strconv"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/idcodec"
	"github.com/benxin_dev/benxinadminpro-server/response"
	"github.com/gin-gonic/gin"
)

// FileHandler 文件管理 handler。
type FileHandler struct {
	svc    *FileService
	errs   *errcode.Registry
	hasher *idcodec.Hasher
	enc    *ResponseEncoder // 始终非 nil；内部容忍 hasher==nil
}

// NewFileHandler 创建文件 handler。
// T-004d：新增 hasher 注入参数（对外契约变更）——消费方装配须注入 idcodec hasher（nil 退化为裸 id）。
func NewFileHandler(svc *FileService, errs *errcode.Registry, hasher *idcodec.Hasher) *FileHandler {
	return &FileHandler{svc: svc, errs: errs, hasher: hasher, enc: NewResponseEncoder(hasher)}
}

// RegisterRoutes 注册文件管理路由（每条经注入的 guard 真 enforce）。
func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup, guard PermGuard) {
	rg.POST("/sys/files", guard.RequirePerm("sys:file:upload"), h.Upload)
	rg.GET("/sys/files/:id/download", guard.RequirePerm("sys:file:download"), h.Download)
	rg.GET("/sys/files", guard.RequirePerm("sys:file:list"), h.List)
	rg.DELETE("/sys/files/:id", guard.RequirePerm("sys:file:delete"), h.Delete)
	// T-011b：批量软删（复用 sys:file:delete 权限码，不新增码）
	rg.POST("/sys/files/batch-delete", guard.RequirePerm("sys:file:delete"), h.BatchDelete)
}

// Upload 上传文件（multipart）。
func (h *FileHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "missing file"})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "open file failed"})
		return
	}
	defer f.Close()

	// 取操作人
	uploader := ""
	if claims := getClaimsFromContext(c); claims != "" {
		uploader = claims
	}

	file, err := h.svc.Upload(
		c.Request.Context(),
		fileHeader.Filename,
		fileHeader.Header.Get("Content-Type"),
		fileHeader.Size,
		f,
		uploader,
	)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, h.enc.Item(file))
}

// Download 鉴权流式下载。
func (h *FileHandler) Download(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil {
		return
	}

	file, reader, err := h.svc.Download(c.Request.Context(), id)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "attachment; filename=\""+file.OriginalName+"\"")
	c.Header("Content-Type", file.Mime)
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))
	c.Status(http.StatusOK)
	// 流式写入，不全载内存
	c.Stream(func(w io.Writer) bool {
		buf := make([]byte, 32*1024)
		_, err := io.CopyBuffer(w, reader, buf)
		return err != nil
	})
}

// List 文件列表。
func (h *FileHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, _ := h.svc.List(c.Request.Context(), FileListQuery{
		UploaderName: c.Query("uploader_name"),
		OriginalName: c.Query("original_name"),
		MimeCategory: c.Query("mime_category"),
		Sort:         c.Query("sort"),
		Order:        c.Query("order"),
		Page:         page,
		PageSize:     ps,
	})
	response.OK(c, h.enc.Page(list, total, page, ps))
}

// BatchDelete 批量软删（入参 hashid 数组，复用 sys:file:delete 权限码）。
// 校验顺序（写前 fail-fast）：空数组/超封顶 → 400(BadReq)；任一非法 hashid → 400(ErrInvalidID，防伪造探测)。
// 幂等：已删/不存在 id 不计入 deleted_count（service IN bulk 取实际命中集）。
func (h *FileHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"` // hashid[]
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadReq(c)
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBatchDeleteIDs {
		response.BadReq(c)
		return
	}
	ids, err := decodeHashSlice(h.hasher, req.IDs)
	if err != nil {
		response.AbortErr(c, h.errs.ErrInvalidID.Code)
		return
	}
	n, err := h.svc.BatchDelete(c.Request.Context(), ids)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, gin.H{"deleted_count": n})
}

// Delete 删除文件。
func (h *FileHandler) Delete(c *gin.Context) {
	id, err := decodePathID(c, h.hasher, h.errs)
	if err != nil {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, nil)
}

func getClaimsFromContext(c *gin.Context) string {
	v, _ := c.Get("jwt_claims")
	if v == nil {
		return ""
	}
	// 避免直接 import auth 包 — 用类型断言取 Subject
	type claimsLike interface{ GetSubject() string }
	if cl, ok := v.(claimsLike); ok {
		return cl.GetSubject()
	}
	return ""
}
