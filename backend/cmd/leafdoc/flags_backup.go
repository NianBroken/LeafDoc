// 本文件声明完整备份与上传恢复参数，启动入口据此组装管理器和调度器。
package main

import (
	"time"

	"github.com/urfave/cli/v3"
)

// backupOptions 保存快照总开关、调度初值、保留数量、目录和上传限额文字。
// 调度设置由设置存储管理，上传大小在启动时转换为字节数。
type backupOptions struct {
	snapshot             bool
	snapshotInterval     time.Duration
	snapshotRetention    int
	snapshotDir          string
	restoreUploadMaxSize string
}

// Flags 绑定命令行与环境变量来源，默认启用快照。目录空值由启动入口
// 展开为数据目录下的 snapshots，不在参数声明阶段创建目录。
func (o *backupOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name: "snapshot",
			// 总开关决定是否创建快照与在线恢复依赖，不等同于周期调度启用标记。
			Destination: &o.snapshot,
			Category:    catSnapshots,
			Usage:       "full backup snapshots (ZIP incl. MySQL records and file resources); disable with --snapshot=false",
			Value:       true,
			DefaultText: "true",
			Sources:     envBoolVars("LEAFDOC_SNAPSHOT"),
		},
		&cli.DurationFlag{
			Name: "snapshot-interval",
			// 间隔提供设置初始化值，已有设置由存储层读取。
			Destination: &o.snapshotInterval,
			Category:    catSnapshots,
			Usage:       "snapshot interval (e.g. 24h, 6h); 0 = manual-only, no automatic scheduling",
			Value:       24 * time.Hour,
			Sources:     envVars("LEAFDOC_SNAPSHOT_INTERVAL"),
			DefaultText: "24h",
		},
		&cli.IntFlag{
			Name:        "snapshot-retention",
			Destination: &o.snapshotRetention,
			Category:    catSnapshots,
			Usage:       "number of most recent snapshots to keep; <= 0 = keep all",
			Value:       10,
			Sources:     envVars("LEAFDOC_SNAPSHOT_RETENTION"),
		},
		&cli.StringFlag{
			Name:        "snapshot-dir",
			Destination: &o.snapshotDir,
			Category:    catSnapshots,
			Usage:       "directory to store snapshot ZIPs in",
			DefaultText: "<data-dir>/snapshots",
			Sources:     envVars("LEAFDOC_SNAPSHOT_DIR"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "restore-upload-max-size",
			// 默认值也接受字节大小校验，拒绝零、无法解析或超出 int64 的值。
			Destination:      &o.restoreUploadMaxSize,
			Category:         catSnapshots,
			Usage:            "maximum size of an uploaded backup ZIP to restore from (e.g. 500MiB, 500MB, 524288000)",
			Value:            "500MiB",
			Sources:          envVars("LEAFDOC_RESTORE_UPLOAD_MAX_SIZE"),
			Validator:        validateByteSizeValue,
			ValidateDefaults: true,
			Config:           trimmed,
		},
	}
}
