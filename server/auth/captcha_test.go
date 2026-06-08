// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   验证码渲染单测 — 字体渲染产出有效 PNG + 字符集排除易混 + 非纯色块
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 15:00:00
// +----------------------------------------------------------------------
package auth

import (
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"
)

// decodeCaptcha 解出渲染结果的 image。
func decodeCaptcha(t *testing.T, dataURI string) image.Image {
	t.Helper()
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURI, prefix) {
		t.Fatalf("captcha not a png data uri: %.30s", dataURI)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURI, prefix))
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	img, err := png.Decode(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	return img
}

// TestCaptchaRendersReadableGlyphs 验证渲染产出有效 PNG，且不是"单一纯色块"——
// 字体绘制会落下一批深色字形像素，背景仍占多数。旧 bug 的伪随机像素块无字形结构。
func TestCaptchaRendersReadableGlyphs(t *testing.T) {
	uri, err := renderCaptchaImage("AB7K")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img := decodeCaptcha(t, uri)

	b := img.Bounds()
	if b.Dx() < 100 || b.Dy() < 40 {
		t.Fatalf("unexpected captcha size: %dx%d", b.Dx(), b.Dy())
	}

	// 统计深色（字形）像素与不同颜色数；既非全同色，也非过半深色
	dark, total := 0, 0
	colors := map[uint32]struct{}{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			total++
			key := (r>>8)<<16 | (g>>8)<<8 | (bl >> 8)
			colors[key] = struct{}{}
			if r>>8 < 110 && g>>8 < 110 && bl>>8 < 140 {
				dark++
			}
		}
	}
	if len(colors) < 5 {
		t.Errorf("captcha looks like a flat block (only %d distinct colors)", len(colors))
	}
	ratio := float64(dark) / float64(total)
	if dark == 0 {
		t.Error("no dark glyph pixels rendered — font drawing failed")
	}
	if ratio > 0.5 {
		t.Errorf("too many dark pixels (%.2f) — not readable glyphs but a block", ratio)
	}
}

// TestCaptchaCharsetExcludesConfusables 字符集不含易混字符 0/O/1/l/I/o。
func TestCaptchaCharsetExcludesConfusables(t *testing.T) {
	for _, ch := range []rune{'0', 'O', '1', 'l', 'I', 'o'} {
		if strings.ContainsRune(defaultCharset, ch) {
			t.Errorf("default charset must exclude confusable %q", ch)
		}
	}
}

// TestGenerateAnswerUsesCharset 生成答案只含给定字符集字符，长度正确。
func TestGenerateAnswerUsesCharset(t *testing.T) {
	const cs = "ABCD2345"
	for i := 0; i < 50; i++ {
		ans := generateAnswer(4, cs)
		if len(ans) != 4 {
			t.Fatalf("answer length = %d, want 4", len(ans))
		}
		for _, ch := range ans {
			if !strings.ContainsRune(cs, ch) {
				t.Fatalf("answer %q contains char %q outside charset", ans, ch)
			}
		}
	}
}

// TestCaptchaSampleDump 落多张样图到 /tmp 供人工目视（不断言，仅辅助验收）。
func TestCaptchaSampleDump(t *testing.T) {
	if os.Getenv("CAPTCHA_DUMP") == "" {
		t.Skip("set CAPTCHA_DUMP=1 to write sample pngs to /tmp/captcha_sample_*.png")
	}
	samples := []string{"Ap76", "K3mN", "wR8q", "7Tb4", "Hx9s", "Ze5g"}
	for i, txt := range samples {
		uri, err := renderCaptchaImage(txt)
		if err != nil {
			t.Fatalf("render %q: %v", txt, err)
		}
		raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, "data:image/png;base64,"))
		path := "/tmp/captcha_sample_" + string(rune('0'+i)) + ".png"
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("wrote %s (%s)", path, txt)
	}
}
