// serviceports 包从嵌入的端口表提供各项服务共用的固定监听端口。
// 端口名称和数值由 ports.json 定义，本包只负责查询、排序和校验，
// 不在查询过程中探测空闲端口，也不自动选择其他端口。
package serviceports

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// allocation 保存构建时嵌入的 JSON 原始字节，查询函数每次从它解析端口表。
// 下方的 go:embed 是编译器识别的资源指令，源码快照中的端口表决定二进制使用的数值。
//
//go:embed ports.json
var allocation []byte

// Names 返回按字典序排列的全部服务名称。
// 每次调用都创建独立切片，调用方修改返回值不会改变端口表。
// 嵌入数据无法解析时直接触发 panic，不返回不完整的名称列表。
func Names() []string {
	var ports map[string]int
	if err := json.Unmarshal(allocation, &ports); err != nil {
		panic(err)
	}
	// map 的遍历次序不固定，收集后排序，使相同配置始终得到相同列表。
	names := make([]string, 0, len(ports))
	for name := range ports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Port 将指定服务的整数端口转换为十进制字符串，供监听参数等字符串接口使用。
// 服务名称不存在时由内部查询触发 panic，不以零端口代替缺失配置。
func Port(service string) string {
	return strconv.Itoa(port(service))
}

// port 解析固定配置并查找精确匹配的名称，不进行大小写或空白归一化。
func port(service string) int {
	var ports map[string]int
	if err := json.Unmarshal(allocation, &ports); err != nil {
		panic(err)
	}
	// 同时检查键是否存在，区分未配置的服务与表中明确保存的整数值。
	value, ok := ports[service]
	if !ok {
		panic("unknown service port: " + service)
	}
	return value
}

// Validator 返回绑定指定服务的字符串校验函数。
// 输入必须与 Port 返回的十进制文本完全一致，否则返回包含固定端口的错误。
// 校验不接受其他数字写法，也不修改传入值或重新分配监听端口。
func Validator(service string) func(string) error {
	return func(value string) error {
		if value != Port(service) {
			return fmt.Errorf("%s must use fixed port %s", service, Port(service))
		}
		return nil
	}
}
