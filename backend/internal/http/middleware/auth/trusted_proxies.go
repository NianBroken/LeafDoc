// 本文件解析代理来源白名单，并将实际网络连接地址与 IP 或网段逐项比较。
// 代理声明的用户身份只有通过来源检查后才由 reverse_proxy.go 消费。
package auth

import (
	"fmt"
	"net"
	"strings"
)

// TrustedProxies 分别保存单个 IP 与 CIDR 网段，供代理身份鉴权检查实际连接来源。
// 集合不包含主机名解析规则，也不读取客户端自行提交的转发请求头。
type TrustedProxies struct {
	nets []*net.IPNet
	ips  []net.IP
}

// ParseTrustedProxies 按逗号分隔配置，去除每项两端空白并忽略空项。
// 含斜杠的项按 CIDR 解析，其余按单个 IP 解析，任一无效项使整体返回错误。
// 空配置得到不信任任何地址的空集合，而非信任全部来源。
func ParseTrustedProxies(raw string) (*TrustedProxies, error) {
	tp := &TrustedProxies{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", entry, err)
			}
			tp.nets = append(tp.nets, ipNet)
		} else {
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP address %q", entry)
			}
			tp.ips = append(tp.ips, ip)
		}
	}
	return tp, nil
}

// IsTrusted 接受带端口地址或裸 IP，先尝试分离端口，再比较单个地址与网段。
// 无法解析为 IP 的文本直接返回 false，不通过 DNS 查询将主机名转成地址。
func (tp *TrustedProxies) IsTrusted(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	// 精确地址和网段任一命中即可通过，两个集合均未命中则拒绝。
	for _, trusted := range tp.ips {
		if trusted.Equal(ip) {
			return true
		}
	}
	for _, ipNet := range tp.nets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}
