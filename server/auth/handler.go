// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   认证 HTTP handler — 登录/刷新/登出/验证码路由
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:18:00
// +----------------------------------------------------------------------

package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/gin-gonic/gin"
)

// Handler 持有认证相关的 HTTP handler。
type Handler struct {
	svc  AuthService
	errs *errcode.Registry
}

// NewHandler 创建认证 handler。
func NewHandler(svc AuthService, errs *errcode.Registry) *Handler {
	return &Handler{svc: svc, errs: errs}
}

// RegisterRoutes 在路由组上注册认证路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/captcha", h.Captcha)
	rg.POST("/auth/login", h.Login)
	rg.POST("/auth/refresh", h.Refresh)
	rg.POST("/auth/logout", h.Logout)
}

// Captcha 获取图形验证码。
func (h *Handler) Captcha(c *gin.Context) {
	captcha, err := h.svc.IssueCaptcha(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondOK(c, captcha)
}

// Login 登录。
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), LoginInput{
		Username:    req.Username,
		Password:    req.Password,
		CaptchaID:   req.CaptchaID,
		CaptchaCode: req.CaptchaCode,
		ClientIP:    c.ClientIP(),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	h.respondOK(c, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"access_exp":    pair.AccessExp,
		"refresh_exp":   pair.RefreshExp,
		"token_type":    "Bearer",
	})
}

// Refresh 刷新令牌。
func (h *Handler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondJSON(c, http.StatusBadRequest, -1, "invalid request", nil)
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.respondError(c, err)
		return
	}

	h.respondOK(c, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"access_exp":    pair.AccessExp,
		"refresh_exp":   pair.RefreshExp,
		"token_type":    "Bearer",
	})
}

// Logout 登出。
func (h *Handler) Logout(c *gin.Context) {
	// access token 从 Authorization header 取
	accessToken := extractBearerToken(c)
	if accessToken == "" {
		h.respondJSON(c, http.StatusUnauthorized, h.errs.ErrTokenInvalid.Code, h.errs.ErrTokenInvalid.Message, nil)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req) // refresh_token 可选

	if err := h.svc.Logout(c.Request.Context(), accessToken, req.RefreshToken); err != nil {
		h.respondError(c, err)
		return
	}

	h.respondOK(c, gin.H{})
}

// ---------------------------------------------------------------------------
// 响应辅助
// ---------------------------------------------------------------------------

func (h *Handler) respondOK(c *gin.Context, data any) {
	h.respondJSON(c, http.StatusOK, 0, "ok", data)
}

func (h *Handler) respondJSON(c *gin.Context, httpStatus, code int, message string, data any) {
	c.JSON(httpStatus, gin.H{
		"code":    code,
		"message": message,
		"data":    data,
	})
}

func (h *Handler) respondError(c *gin.Context, err error) {
	var ecErr errcode.Error
	if errors.As(err, &ecErr) {
		h.respondJSON(c, ecErr.HTTP, ecErr.Code, ecErr.Message, nil)
		return
	}
	h.respondJSON(c, http.StatusInternalServerError, -1, "internal error", nil)
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return auth[7:]
	}
	return ""
}
