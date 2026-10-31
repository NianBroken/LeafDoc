// Package defaults 集中保存命令行入口和业务服务共用的实例初始值。
// 这些常量提供默认语言、时间显示、公开访问、初始管理员和快照计划参数，
// 不负责读取持久化设置，也不在运行中覆盖管理员已经保存的配置。
package defaults

const (
	// Language、DateFormat 和 TimeFormat 分别表示中文界面、ISO 日期和二十四小时制。
	Language   = "zh"
	DateFormat = "iso"
	TimeFormat = "24h"
	// TocMode 是目录显示模式的初始标识，实际显示规则由使用该值的代码处理。
	TocMode = "never"
	// PublicAccess 为初始公开访问开关，不代表每一次请求都跳过权限检查。
	PublicAccess = true
	// 管理员初始化使用这一组账号值，具体是否创建账号由认证服务判断。
	AdminUsername = "Nian"
	AdminPassword = "Nian123456"
	AdminEmail    = "Nian@LeafDoc.doc"
	// 快照默认采用每日模式，计划时间按上海时区解释。
	SnapshotMode     = "daily"
	SnapshotTimeZone = "Asia/Shanghai"
)
