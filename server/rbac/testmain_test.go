// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   rbac 测试初始化 — response.Registry 全局初始化
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-08 02:52:00
// +----------------------------------------------------------------------

package rbac

import (
	"os"
	"testing"

	"github.com/benxin_dev/benxinadminpro-server/errcode"
	"github.com/benxin_dev/benxinadminpro-server/response"
)

func TestMain(m *testing.M) {
	reg, _ := errcode.NewRegistry(11000)
	response.InitFromErrcode(reg.AllSpecs())
	os.Exit(m.Run())
}
