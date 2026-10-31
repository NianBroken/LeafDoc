// 本文件定义恢复管理器所需的资源与回调，不读取配置或执行恢复操作。
package restore

import (
	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/settings"
	"github.com/NianBroken/LeafDoc/internal/core/snapshot"
)

// Config 组合快照定位、数据目录、写入屏障和运行态重载入口。
// ReloadRuntime 存在时优先使用它，否则才调用 TriggerResync；
// MaxUploadSizeBytes 供上传入口读取限额，管理器的磁盘暂存方法本身不执行限额裁剪。
type Config struct {
	SnapshotManager    *snapshot.Manager
	DataDir            string
	WikiVersion        string
	WriteGate          *WriteGate
	AuthService        *auth.AuthService
	Reloadables        []settings.Reloadable
	UserResolver       *auth.UserResolver
	TriggerResync      func()
	ReloadRuntime      func() error
	MaxUploadSizeBytes int64
}
