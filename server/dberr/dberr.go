// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   DB 错误归一化中立包 — 唯一键冲突（MySQL 1062）检测单一出入口
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-09 17:00:00
// +----------------------------------------------------------------------
//
// 本包是业务中立工具：仅识别"DB 唯一键冲突"这一类错误，供各 service 在写操作
// 边界把 1062 转成自己实体的友好业务错误码（如"用户名已存在"）。
// 设计同 idcodec：rbac/system 共用、互不 import，依赖仅 mysql 驱动 + gorm。
// 单一出入口：1062 / ErrDuplicatedKey 的判定只此一处，杜绝裸 Number==1062 散落各处。

package dberr

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// mysqlDupEntry 是 MySQL "Duplicate entry"（唯一键冲突）的错误号。
// 参见 MySQL Error 1062 / ER_DUP_ENTRY。
const mysqlDupEntry = 1062

// IsDuplicate 报告 err 是否为 DB 唯一键冲突。
// 兼容两条路径：① GORM 开启 TranslateError 时归一化的 gorm.ErrDuplicatedKey；
// ② 未开启时直接冒泡的 *mysql.MySQLError（Number==1062）。
// nil 或其它错误（外键 1452、连接失败等）返回 false——不误转，调用方仍走原错误路径。
func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == mysqlDupEntry
	}
	return false
}
