// 本文件声明独立指标监听参数，端口使用共享分配规则并在解析阶段校验。
package main

import (
	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
	"github.com/urfave/cli/v3"
)

// metricsOptions 保存指标启用标记与监听地址，不复用主业务 HTTP 服务。
type metricsOptions struct {
	enableMetrics bool
	metricsHost   string
	metricsPort   string
}

// Flags 将开关、主机和固定指标端口绑定到目标字段。
// 默认监听回环地址；端口默认值和显式输入都经过同一个共享校验器。
func (o *metricsOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "enable-metrics",
			Destination: &o.enableMetrics,
			Category:    catMetrics,
			Usage:       "enable the Prometheus /metrics endpoint on a separate listener",
			Sources:     envBoolVars("LEAFDOC_ENABLE_METRICS"),
		},
		&cli.StringFlag{
			Name:        "metrics-host",
			Destination: &o.metricsHost,
			Category:    catMetrics,
			Usage:       "host/IP address for the Prometheus metrics listener",
			Value:       "127.0.0.1",
			Sources:     envVars("LEAFDOC_METRICS_HOST"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "metrics-port",
			// 端口从 serviceports 读取，拒绝与统一分配不同的覆盖值。
			Destination:      &o.metricsPort,
			Category:         catMetrics,
			Usage:            "port for the Prometheus metrics listener",
			Value:            serviceports.Port("metrics"),
			Validator:        serviceports.Validator("metrics"),
			ValidateDefaults: true,
			Sources:          envVars("LEAFDOC_METRICS_PORT"),
			Config:           trimmed,
		},
	}
}
