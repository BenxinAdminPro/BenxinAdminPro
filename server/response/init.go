// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   response 初始化辅助 — 从 errcode.Registry 注册全部码到 Renderer
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 02:50:00
// +----------------------------------------------------------------------

package response

import "fmt"

// InitFromErrcode 从 errcode.Registry.AllSpecs() 注册全部码并设置全局 Renderer。
// 启动和测试初始化时调用。
func InitFromErrcode(allSpecs []struct{ Code, HTTP int; I18nKey string }) error {
	reg := NewRegistry()
	for _, s := range allSpecs {
		if err := reg.Register(ErrSpec{Code: s.Code, HTTP: s.HTTP, I18nKey: s.I18nKey}); err != nil {
			return fmt.Errorf("response init: %w", err)
		}
	}
	renderer := NewRenderer(reg, nil)
	SetDefault(renderer)
	return nil
}
