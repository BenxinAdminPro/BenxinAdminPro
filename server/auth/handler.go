// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   认证 HTTP handler — 登录/刷新/登出/验证码路由
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:18:00
// | @updated   2026-06-08 02:40:00
// | @updated   2026-06-08 15:00:00  T-002b：新增 /auth/precheck（前端按需显示验证码的服务端信号）
// +----------------------------------------------------------------------

package auth

import (
	"strings"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
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
	rg.POST("/auth/precheck", h.Precheck)
	rg.POST("/auth/login", h.Login)
	rg.POST("/auth/refresh", h.Refresh)
	rg.POST("/auth/logout", h.Logout)
}

// Captcha 获取图形验证码。
func (h *Handler) Captcha(c *gin.Context) {
	captcha, err := h.svc.IssueCaptcha(c.Request.Context())
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, captcha)
}

// Precheck 返回该用户名当前是否需要验证码（前端据此按需显示）。
// 注意：这是 UX 信号，登录是否强制校验由后端 Login 独立判定，前端不可绕过。
func (h *Handler) Precheck(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadReq(c)
		return
	}
	required, err := h.svc.Precheck(c.Request.Context(), req.Username)
	if err != nil {
		response.ErrResp(c, err)
		return
	}
	response.OK(c, gin.H{"captcha_required": required})
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
		response.BadReq(c)
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
		response.ErrResp(c, err)
		return
	}

	response.OK(c, gin.H{
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
		response.BadReq(c)
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.ErrResp(c, err)
		return
	}

	response.OK(c, gin.H{
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
		response.AbortErr(c, h.errs.ErrTokenInvalid.Code)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req) // refresh_token 可选

	if err := h.svc.Logout(c.Request.Context(), accessToken, req.RefreshToken); err != nil {
		response.ErrResp(c, err)
		return
	}

	response.OK(c, gin.H{})
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return auth[7:]
	}
	return ""
}
