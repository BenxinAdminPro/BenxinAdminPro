// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   文件管理 handler — 上传/下载/列表/删除
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 01:25:00
// +----------------------------------------------------------------------

package system

import (
	"io"
	"net/http"
	"strconv"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

// FileHandler 文件管理 handler。
type FileHandler struct {
	svc  *FileService
	errs *errcode.Registry
}

// NewFileHandler 创建文件 handler。
func NewFileHandler(svc *FileService, errs *errcode.Registry) *FileHandler {
	return &FileHandler{svc: svc, errs: errs}
}

// RegisterRoutes 注册文件管理路由。
func (h *FileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/sys/files", requirePerm("sys:file:upload"), h.Upload)
	rg.GET("/sys/files/:id/download", requirePerm("sys:file:download"), h.Download)
	rg.GET("/sys/files", requirePerm("sys:file:list"), h.List)
	rg.DELETE("/sys/files/:id", requirePerm("sys:file:delete"), h.Delete)
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
		fail(c, err)
		return
	}
	ok(c, file)
}

// Download 鉴权流式下载。
func (h *FileHandler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		bad(c)
		return
	}

	file, reader, err := h.svc.Download(c.Request.Context(), id)
	if err != nil {
		fail(c, err)
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
	uploader := c.Query("uploader")
	list, total, _ := h.svc.List(c.Request.Context(), uploader, page, ps)
	ok(c, gin.H{"list": list, "total": total, "page": page, "page_size": ps})
}

// Delete 删除文件。
func (h *FileHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		bad(c)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
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
