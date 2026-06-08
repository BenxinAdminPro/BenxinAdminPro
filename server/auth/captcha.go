// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   图形验证码 — 生成 + 一次性校验 + CaptchaStore 接口
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 17:06:00
// | @updated   2026-06-08 15:00:00  T-002b：换开源 Go 字体(opentype)渲染清晰字符，替换不可读的像素块；字符集可配
// +----------------------------------------------------------------------

package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/big"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
)

// captchaFont 是验证码渲染字体（Go Mono Bold，BSD-3-Clause 开源字体），启动时解析一次。
var captchaFont *opentype.Font

func init() {
	f, err := opentype.Parse(gomonobold.TTF)
	if err != nil {
		// 字体经 Go module 内嵌、构建期确定，解析失败属构建/依赖损坏，panic 暴露
		panic("auth: parse captcha font: " + err.Error())
	}
	captchaFont = f
}

// Captcha 表示生成的验证码结果。
type Captcha struct {
	CaptchaID   string `json:"captcha_id"`
	ImageBase64 string `json:"image_base64"` // data:image/png;base64,...
	ExpiresIn   int    `json:"expires_in"`   // 秒
}

// CaptchaConfig 验证码配置。
type CaptchaConfig struct {
	Enabled       bool   // 是否启用
	TTLSeconds    int    // 答案 TTL，默认 120
	Length        int    // 验证码位数，默认 4
	CaseSensitive bool   // 大小写敏感
	Charset       string // 字符集（默认排除易混字符 0/O/1/l/I）；可配置注入
}

// defaultCharset 默认字符集：已排除易混字符（无 0/O、1/l/I）。
const defaultCharset = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"

func (c CaptchaConfig) charset() string {
	if c.Charset == "" {
		return defaultCharset
	}
	return c.Charset
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

	answer := generateAnswer(s.cfg.length(), s.cfg.charset())

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

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateAnswer(length int, charset string) string {
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// randInt 返回 [0,n) 的随机整数（仅用于视觉抖动/干扰，非安全敏感）。
func randInt(n int) int {
	if n <= 0 {
		return 0
	}
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(v.Int64())
}

// renderCaptchaImage 用开源 Go 字体（opentype）渲染验证码 PNG。
// 防自动化：逐字符小角度旋转 + 整体波浪扭曲 + 间距收紧略粘连 + 交叉干扰线 + 噪点。
// 可读性：保持人眼稍辨认即可读（"看一眼能读出但不一目了然"），不退化为不可读色块。
// 返回 data:image/png;base64,... 格式。
func renderCaptchaImage(text string) (string, error) {
	const (
		height   = 50
		fontSize = 30.0
		cellStep = 24 // 每字符水平步进（收紧 → 略粘连）
		cellW    = 40 // 单字符画布宽（大于步进，留旋转溢出 + 重叠）
		padX     = 10
	)
	width := cellStep*len(text) + padX*2 + 6
	if width < 120 {
		width = 120
	}

	face, err := opentype.NewFace(captchaFont, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return "", err
	}
	defer face.Close()

	// 文本层（透明）：逐字符旋转后合成，便于整体波浪扭曲
	layer := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, ch := range text {
		// 字符保持较深以确保对比度（抗 OCR 靠扭曲/干扰线，不靠淡色）
		col := color.RGBA{uint8(15 + randInt(50)), uint8(15 + randInt(50)), uint8(55 + randInt(85)), 255}
		cell := drawGlyphCell(face, string(ch), cellW, height, col)
		angle := float64(randInt(33)-16) * math.Pi / 180 // ±16°
		dstX := padX + i*cellStep + randInt(5) - 2
		dstY := randInt(8) - 4
		compositeRotated(layer, cell, angle, dstX, dstY)
	}
	addLines(layer, width, height) // 干扰线进文本层，随波浪一起扭曲

	warped := warpWave(layer, width, height)

	// 主图：浅色背景 + 合成扭曲后的文本层 + 顶层噪点
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bg := color.RGBA{uint8(232 + randInt(20)), uint8(232 + randInt(20)), uint8(236 + randInt(18)), 255}
	xdraw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, xdraw.Src)
	xdraw.Draw(img, img.Bounds(), warped, image.Point{}, xdraw.Over)
	addNoiseDots(img, width, height)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// drawGlyphCell 在透明小画布上绘制单个字符（供后续旋转/合成）。
func drawGlyphCell(face font.Face, ch string, cellW, height int, col color.RGBA) *image.RGBA {
	cell := image.NewRGBA(image.Rect(0, 0, cellW, height))
	d := &font.Drawer{Dst: cell, Src: &image.Uniform{col}, Face: face}
	d.Dot = fixed.Point26_6{X: fixed.I(8), Y: fixed.I(36)}
	d.DrawString(ch)
	return cell
}

// compositeRotated 把 cell 绕中心旋转 angle 后合成到 dst 的 (dstX,dstY)。
func compositeRotated(dst *image.RGBA, cell *image.RGBA, angle float64, dstX, dstY int) {
	b := cell.Bounds()
	cx, cy := float64(b.Dx())/2, float64(b.Dy())/2
	cos, sin := math.Cos(angle), math.Sin(angle)
	// 绕中心旋转后平移到目标位置
	m := f64.Aff3{
		cos, -sin, cx - cos*cx + sin*cy + float64(dstX),
		sin, cos, cy - sin*cx - cos*cy + float64(dstY),
	}
	xdraw.CatmullRom.Transform(dst, m, cell, b, xdraw.Over, nil)
}

// warpWave 对整图做列向正弦位移（波浪扭曲），随机相位/振幅。
func warpWave(src *image.RGBA, width, height int) *image.RGBA {
	out := image.NewRGBA(src.Bounds())
	amp := 3.5 + float64(randInt(20))/10 // 3.5~5.5px
	period := float64(width)/1.6 + float64(randInt(20))
	phase := float64(randInt(628)) / 100 // 0~2π
	for x := 0; x < width; x++ {
		dy := int(math.Round(amp * math.Sin(2*math.Pi*float64(x)/period+phase)))
		for y := 0; y < height; y++ {
			sy := y - dy
			if sy >= 0 && sy < height {
				out.Set(x, y, src.At(x, sy))
			}
		}
	}
	return out
}

// addLines 叠加交叉干扰线（部分较深，穿过字符）。
func addLines(img *image.RGBA, width, height int) {
	for i := 0; i < 5; i++ {
		// 半数深色干扰线（更贴近字符颜色，增加切割难度）
		var lc color.RGBA
		if i%2 == 0 {
			lc = color.RGBA{uint8(60 + randInt(70)), uint8(60 + randInt(70)), uint8(90 + randInt(80)), 255}
		} else {
			lc = color.RGBA{uint8(150 + randInt(60)), uint8(150 + randInt(60)), uint8(150 + randInt(60)), 255}
		}
		y1, y2 := randInt(height), randInt(height)
		for x := 0; x < width; x++ {
			y := y1 + (y2-y1)*x/width
			// 线宽 1~2px
			for t := 0; t <= randInt(2); t++ {
				yy := y + t
				if yy >= 0 && yy < height {
					img.Set(x, yy, lc)
				}
			}
		}
	}
}

// addNoiseDots 叠加噪点（约 3% 像素）。
func addNoiseDots(img *image.RGBA, width, height int) {
	dots := width * height / 32
	for i := 0; i < dots; i++ {
		c := color.RGBA{uint8(90 + randInt(120)), uint8(90 + randInt(120)), uint8(90 + randInt(120)), 255}
		img.Set(randInt(width), randInt(height), c)
	}
}
