// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   图形验证码 — 生成 + 一次性校验 + CaptchaStore 接口
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:06:00
// +----------------------------------------------------------------------

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/big"
	"strings"
	"time"

	"bytes"
)

// Captcha 表示生成的验证码结果。
type Captcha struct {
	CaptchaID   string `json:"captcha_id"`
	ImageBase64 string `json:"image_base64"` // data:image/png;base64,...
	ExpiresIn   int    `json:"expires_in"`   // 秒
}

// CaptchaConfig 验证码配置。
type CaptchaConfig struct {
	Enabled       bool  // 是否启用
	TTLSeconds    int   // 答案 TTL，默认 120
	Length        int   // 验证码位数，默认 4
	CaseSensitive bool  // 大小写敏感
}

func (c CaptchaConfig) ttl() time.Duration {
	if c.TTLSeconds <= 0 {
		return 120 * time.Second
	}
	return time.Duration(c.TTLSeconds) * time.Second
}

func (c CaptchaConfig) length() int {
	if c.Length <= 0 {
		return 4
	}
	return c.Length
}

// CaptchaStore 抽象验证码答案存储。
type CaptchaStore interface {
	// Set 存储验证码答案，TTL 到期后自动过期。
	Set(ctx context.Context, key, answer string, ttl time.Duration) error
	// GetAndDelete 获取并删除答案（一次性消费，无论对错都删）。
	// 不存在或已过期返回 "", nil。
	GetAndDelete(ctx context.Context, key string) (string, error)
}

// CaptchaService 验证码服务。
type CaptchaService struct {
	store          CaptchaStore
	cfg            CaptchaConfig
	redisKeyPrefix string
}

// NewCaptchaService 创建验证码服务。
func NewCaptchaService(store CaptchaStore, cfg CaptchaConfig, redisKeyPrefix string) *CaptchaService {
	return &CaptchaService{store: store, cfg: cfg, redisKeyPrefix: redisKeyPrefix}
}

func (s *CaptchaService) captchaKey(id string) string {
	prefix := s.redisKeyPrefix
	if prefix == "" {
		prefix = "app"
	}
	return prefix + ":auth:captcha:" + id
}

// Generate 生成验证码：返回 captcha_id + base64 PNG + 过期秒数。
func (s *CaptchaService) Generate(ctx context.Context) (Captcha, error) {
	id, err := generateID()
	if err != nil {
		return Captcha{}, fmt.Errorf("auth: generate captcha id: %w", err)
	}

	answer := generateAnswer(s.cfg.length())

	// 存储答案
	key := s.captchaKey(id)
	if err := s.store.Set(ctx, key, answer, s.cfg.ttl()); err != nil {
		return Captcha{}, fmt.Errorf("auth: store captcha: %w", err)
	}

	// 生成图片
	imgB64, err := renderCaptchaImage(answer)
	if err != nil {
		return Captcha{}, fmt.Errorf("auth: render captcha: %w", err)
	}

	return Captcha{
		CaptchaID:   id,
		ImageBase64: imgB64,
		ExpiresIn:   int(s.cfg.ttl().Seconds()),
	}, nil
}

// Verify 校验验证码（一次性消费：校验即删，无论对错都删，防爆破）。
func (s *CaptchaService) Verify(ctx context.Context, captchaID, code string) (bool, error) {
	key := s.captchaKey(captchaID)
	answer, err := s.store.GetAndDelete(ctx, key)
	if err != nil {
		return false, err
	}
	if answer == "" {
		return false, nil // 不存在或已过期
	}

	if s.cfg.CaseSensitive {
		return answer == code, nil
	}
	return strings.EqualFold(answer, code), nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateAnswer(length int) string {
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// renderCaptchaImage 生成简单的验证码 PNG 图片。
// 返回 data:image/png;base64,... 格式。
func renderCaptchaImage(text string) (string, error) {
	width := 40 * len(text)
	if width < 120 {
		width = 120
	}
	height := 50

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 浅色背景
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{240, 240, 240, 255})
		}
	}

	// 简单的字符绘制（每个字符用块表示）
	charWidth := width / len(text)
	for i, ch := range text {
		drawChar(img, i*charWidth+5, 10, byte(ch))
	}

	// 添加干扰线
	addNoise(img, width, height)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + b64, nil
}

// drawChar 用简单像素块画一个字符。
func drawChar(img *image.RGBA, x, y int, ch byte) {
	c := color.RGBA{uint8(50 + ch%100), uint8(30 + ch%80), uint8(80 + ch%120), 255}
	// 简单的 5x7 像素块代表字符
	for dy := 0; dy < 28; dy++ {
		for dx := 0; dx < 20; dx++ {
			// 基于字符值产生伪随机图案
			if (int(ch)*7+dx*3+dy*5)%4 != 0 {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}

// addNoise 添加干扰线。
func addNoise(img *image.RGBA, width, height int) {
	noiseColor := color.RGBA{180, 180, 180, 255}
	for i := 0; i < 3; i++ {
		y1B, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y2B, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		y1, y2 := int(y1B.Int64()), int(y2B.Int64())
		for x := 0; x < width; x++ {
			y := y1 + (y2-y1)*x/width
			if y >= 0 && y < height {
				img.Set(x, y, noiseColor)
			}
		}
	}
}
