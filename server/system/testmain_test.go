// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   system 测试初始化 — response.Registry 全局初始化（T-004d：handler 400 路径依赖渲染器）
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 16:09:17
// +----------------------------------------------------------------------

package system

import (
	"os"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
)

func TestMain(m *testing.M) {
	r, _ := errcode.NewRegistry(11000)
	response.InitFromErrcode(r.AllSpecs())
	os.Exit(m.Run())
}
