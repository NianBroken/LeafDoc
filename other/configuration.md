# 配置与 YAML 逐字段参考

## 配置来源

| 来源 | 使用方式 | 生效 |
| --- | --- | --- |
| Compose 与环境替代 | 部署入口读取 environment，解析容器定义 | compose up -d 重建受影响容器 |
| application.env | 离线包 app 的 env_file | 重新创建 app |
| 安装记录与 MySQL | 向导保存数据库和安装状态 | 依项目安装及服务机制加载，不是编辑 YAML |
| 管理员设置 | 通过实际设置 API 保存 | 按对应服务读取机制生效 |

## 表格规则

| 项目 | 规则 |
| --- | --- |
| 当前值 | 文件里实际声明，不冒充程序默认值 |
| 默认 | ${VAR:-x} 表示替代默认 x，${VAR:?} 表示缺失时解析失败；无替代不代表工具没有自身默认 |
| 填写类别 | 必须填写指部署来源必须提供；不需要填写指模板或工具固定项；可选指确有可用默认或特定自定义入口 |
| 生效 | Compose 改动执行对应部署入口重新应用，lint 改动在下次检查生效 |
| YAML 别名 | 解析表展开别名，源文件 & 定义和 * 引用共用同一结构；<< 合并后的每个字段仍单列 |

## other/.golangci.yml

| 所属文件 | 完整路径 | 当前值 | 含义、作用、场景、目的及关联限制 | 填写要求 | 填写格式与内容 | 默认和省略行为 | 修改生效及影响 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| .golangci.yml | version | 2 | golangci-lint 配置格式版本，2 对应项目固定检查器配置格式 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.enable | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.enable[0] | govet | 显式启用 govet，检查或格式化规则由固定工具版本解释 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.enable[1] | unused | 显式启用 unused，检查或格式化规则由固定工具版本解释 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.enable[2] | gocritic | 显式启用 gocritic，检查或格式化规则由固定工具版本解释 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.enable[3] | staticcheck | 显式启用 staticcheck，检查或格式化规则由固定工具版本解释 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.exclusions | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.exclusions.generated | lax | lax 使用宽松的生成文件识别规则，不把它用于隐藏手写代码问题 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.exclusions.paths | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.exclusions.paths[0] | third_party$ | 检查器排除路径表达式 third_party$，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.exclusions.paths[1] | builtin$ | 检查器排除路径表达式 builtin$，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.exclusions.paths[2] | examples$ | 检查器排除路径表达式 examples$，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | linters.exclusions.paths[3] | node_modules | 检查器排除路径表达式 node_modules，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.enable | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.enable[0] | gofmt | 显式启用 gofmt，检查或格式化规则由固定工具版本解释 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.enable[1] | goimports | 显式启用 goimports，检查或格式化规则由固定工具版本解释 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.exclusions | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.exclusions.generated | lax | lax 使用宽松的生成文件识别规则，不把它用于隐藏手写代码问题 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.exclusions.paths | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.exclusions.paths[0] | third_party$ | 检查器排除路径表达式 third_party$，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.exclusions.paths[1] | builtin$ | 检查器排除路径表达式 builtin$，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.exclusions.paths[2] | examples$ | 检查器排除路径表达式 examples$，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |
| .golangci.yml | formatters.exclusions.paths[3] | node_modules | 检查器排除路径表达式 node_modules，作用于对应 linters 或 formatters 区域 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | 下次 lint 检查加载 |

### 必须填写

| 完整路径 | 要求与原因 |
| --- | --- |
| 无 | 本文件没有此类字段 |

### 不需要填写

| 完整路径 | 要求与原因 |
| --- | --- |
| version | golangci-lint 配置格式版本，2 对应项目固定检查器配置格式 |
| linters | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| linters.enable | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| linters.enable[0] | 显式启用 govet，检查或格式化规则由固定工具版本解释 |
| linters.enable[1] | 显式启用 unused，检查或格式化规则由固定工具版本解释 |
| linters.enable[2] | 显式启用 gocritic，检查或格式化规则由固定工具版本解释 |
| linters.enable[3] | 显式启用 staticcheck，检查或格式化规则由固定工具版本解释 |
| linters.exclusions | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| linters.exclusions.generated | lax 使用宽松的生成文件识别规则，不把它用于隐藏手写代码问题 |
| linters.exclusions.paths | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| linters.exclusions.paths[0] | 检查器排除路径表达式 third_party$，作用于对应 linters 或 formatters 区域 |
| linters.exclusions.paths[1] | 检查器排除路径表达式 builtin$，作用于对应 linters 或 formatters 区域 |
| linters.exclusions.paths[2] | 检查器排除路径表达式 examples$，作用于对应 linters 或 formatters 区域 |
| linters.exclusions.paths[3] | 检查器排除路径表达式 node_modules，作用于对应 linters 或 formatters 区域 |
| formatters | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| formatters.enable | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| formatters.enable[0] | 显式启用 gofmt，检查或格式化规则由固定工具版本解释 |
| formatters.enable[1] | 显式启用 goimports，检查或格式化规则由固定工具版本解释 |
| formatters.exclusions | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| formatters.exclusions.generated | lax 使用宽松的生成文件识别规则，不把它用于隐藏手写代码问题 |
| formatters.exclusions.paths | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| formatters.exclusions.paths[0] | 检查器排除路径表达式 third_party$，作用于对应 linters 或 formatters 区域 |
| formatters.exclusions.paths[1] | 检查器排除路径表达式 builtin$，作用于对应 linters 或 formatters 区域 |
| formatters.exclusions.paths[2] | 检查器排除路径表达式 examples$，作用于对应 linters 或 formatters 区域 |
| formatters.exclusions.paths[3] | 检查器排除路径表达式 node_modules，作用于对应 linters 或 formatters 区域 |

### 可以填写也可以不填写

| 完整路径 | 要求与原因 |
| --- | --- |
| 无 | 本文件没有此类字段 |

## other/compose.bundle.yaml

| 所属文件 | 完整路径 | 当前值 | 含义、作用、场景、目的及关联限制 | 填写要求 | 填写格式与内容 | 默认和省略行为 | 修改生效及影响 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| compose.bundle.yaml | services | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.image | ${LEAFDOC_IMAGE:?} | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_IMAGE 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.pull_policy | never | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.restart | unless-stopped | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.env_file | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.env_file[0] | /etc/leafdoc/application.env | 容器运行环境文件，宿主实际路径必须存在，修改后重新创建容器 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.environment.LEAFDOC_HOST | 0.0.0.0 | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.environment.LEAFDOC_TLS_CERT | /run/leafdoc-tls/server.crt | 容器内服务证书路径，须与只读 TLS 挂载一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.environment.LEAFDOC_TLS_KEY | /run/leafdoc-tls/server.key | 容器内证书私钥路径，需可读并与证书匹配 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.environment.LEAFDOC_WEB_SETUP | true | 开启未安装实例的网页安装入口，已安装状态仍按持久记录判断 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.environment.LEAFDOC_DATA_DIR | /app/data | 容器内实例文件位置，必须与持久挂载对应，不是 MySQL 数据目录 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.environment.LEAFDOC_BASE_PATH | ${LEAFDOC_BASE_PATH:-} | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BASE_PATH 缺失时使用 空字符串 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.ports | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.ports[0] | ${LEAFDOC_BIND_ADDRESS:-0.0.0.0}:${LEAFDOC_BACKEND_PORT}:${LEAFDOC_BACKEND_PORT}/tcp | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BIND_ADDRESS 缺失时使用 0.0.0.0；LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.command | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.command[0] | --enable-revision | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.command[1] | --snapshot | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.volumes[0] | ${LEAFDOC_CONFIG_DIR:-/etc/leafdoc}/tls:/run/leafdoc-tls:ro | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_CONFIG_DIR 缺失时使用 /etc/leafdoc | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.volumes[1] | /var/lib/leafdoc:/app/data | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.healthcheck | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.healthcheck.test | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.healthcheck.test[0] | CMD-SHELL | 在容器 shell 中执行探测，支持环境变量展开 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.healthcheck.test[1] | wget -qO- "http://127.0.0.1:${LEAFDOC_BACKEND_PORT}$${LEAFDOC_BASE_PATH}/api/health" > /dev/null | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_BASE_PATH 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.healthcheck.interval | 10s | 健康探测间隔，持续时间格式，影响检查频率 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.healthcheck.timeout | 5s | 单次健康探测超时，超过时本轮失败 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.healthcheck.retries | 12 | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 | 不需要填写 | 整数；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.app.stop_grace_period | 45s | 优雅停止等待时间，超时后 Docker 强制结束进程 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.image | ${LEAFDOC_IMAGE:-leafdoc:local} | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_IMAGE 缺失时使用 leafdoc:local | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.restart | unless-stopped | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.command | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.command[0] | gateway | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.environment.LEAFDOC_HOST | 0.0.0.0 | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.environment.LEAFDOC_BASE_PATH | ${LEAFDOC_BASE_PATH:-} | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BASE_PATH 缺失时使用 空字符串 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.environment.LEAFDOC_BACKEND_URL | http://app:${LEAFDOC_BACKEND_PORT} | 网关转发目标，服务名 app 只在对应 Compose 网络内解析 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.environment.LEAFDOC_TLS_CERT | /run/leafdoc-tls/server.crt | 容器内服务证书路径，须与只读 TLS 挂载一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.environment.LEAFDOC_TLS_KEY | /run/leafdoc-tls/server.key | 容器内证书私钥路径，需可读并与证书匹配 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.depends_on | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.depends_on[0] | app | 依赖服务名或条件，不是容器外部数据库地址；简单列表不等待健康成功 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.ports | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.ports[0] | ${LEAFDOC_BIND_ADDRESS:-0.0.0.0}:${LEAFDOC_FRONTEND_PORT}:${LEAFDOC_FRONTEND_PORT}/tcp | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BIND_ADDRESS 缺失时使用 0.0.0.0；LEAFDOC_FRONTEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_FRONTEND_PORT 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.volumes[0] | ${LEAFDOC_CONFIG_DIR:-/etc/leafdoc}/tls:/run/leafdoc-tls:ro | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_CONFIG_DIR 缺失时使用 /etc/leafdoc | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.healthcheck | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.healthcheck.test | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.healthcheck.test[0] | CMD-SHELL | 在容器 shell 中执行探测，支持环境变量展开 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.healthcheck.test[1] | wget -qO- "http://127.0.0.1:${LEAFDOC_FRONTEND_PORT}$${LEAFDOC_BASE_PATH}/api/health" > /dev/null | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_FRONTEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_BASE_PATH 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.healthcheck.interval | 10s | 健康探测间隔，持续时间格式，影响检查频率 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.healthcheck.timeout | 5s | 单次健康探测超时，超过时本轮失败 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.healthcheck.retries | 12 | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 | 不需要填写 | 整数；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.bundle.yaml | services.gateway.stop_grace_period | 30s | 优雅停止等待时间，超时后 Docker 强制结束进程 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |

### 必须填写

| 完整路径 | 要求与原因 |
| --- | --- |
| services.app.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |

### 不需要填写

| 完整路径 | 要求与原因 |
| --- | --- |
| services | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.pull_policy | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 |
| services.app.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.app.env_file | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.env_file[0] | 容器运行环境文件，宿主实际路径必须存在，修改后重新创建容器 |
| services.app.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.environment.LEAFDOC_HOST | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 |
| services.app.environment.LEAFDOC_TLS_CERT | 容器内服务证书路径，须与只读 TLS 挂载一致 |
| services.app.environment.LEAFDOC_TLS_KEY | 容器内证书私钥路径，需可读并与证书匹配 |
| services.app.environment.LEAFDOC_WEB_SETUP | 开启未安装实例的网页安装入口，已安装状态仍按持久记录判断 |
| services.app.environment.LEAFDOC_DATA_DIR | 容器内实例文件位置，必须与持久挂载对应，不是 MySQL 数据目录 |
| services.app.ports | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.command | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.command[0] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.app.command[1] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.app.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.volumes[1] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.app.healthcheck | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.healthcheck.test | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.healthcheck.test[0] | 在容器 shell 中执行探测，支持环境变量展开 |
| services.app.healthcheck.test[1] | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 |
| services.app.healthcheck.interval | 健康探测间隔，持续时间格式，影响检查频率 |
| services.app.healthcheck.timeout | 单次健康探测超时，超过时本轮失败 |
| services.app.healthcheck.retries | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 |
| services.app.stop_grace_period | 优雅停止等待时间，超时后 Docker 强制结束进程 |
| services.gateway | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.gateway.command | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.command[0] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.gateway.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.environment.LEAFDOC_HOST | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 |
| services.gateway.environment.LEAFDOC_BACKEND_URL | 网关转发目标，服务名 app 只在对应 Compose 网络内解析 |
| services.gateway.environment.LEAFDOC_TLS_CERT | 容器内服务证书路径，须与只读 TLS 挂载一致 |
| services.gateway.environment.LEAFDOC_TLS_KEY | 容器内证书私钥路径，需可读并与证书匹配 |
| services.gateway.depends_on | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.depends_on[0] | 依赖服务名或条件，不是容器外部数据库地址；简单列表不等待健康成功 |
| services.gateway.ports | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.healthcheck | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.healthcheck.test | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.healthcheck.test[0] | 在容器 shell 中执行探测，支持环境变量展开 |
| services.gateway.healthcheck.test[1] | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 |
| services.gateway.healthcheck.interval | 健康探测间隔，持续时间格式，影响检查频率 |
| services.gateway.healthcheck.timeout | 单次健康探测超时，超过时本轮失败 |
| services.gateway.healthcheck.retries | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 |
| services.gateway.stop_grace_period | 优雅停止等待时间，超时后 Docker 强制结束进程 |

### 可以填写也可以不填写

| 完整路径 | 要求与原因 |
| --- | --- |
| services.app.environment.LEAFDOC_BASE_PATH | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 |
| services.app.ports[0] | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| services.app.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.gateway.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |
| services.gateway.environment.LEAFDOC_BASE_PATH | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 |
| services.gateway.ports[0] | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| services.gateway.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |

## other/compose.fnos.yaml

| 所属文件 | 完整路径 | 当前值 | 含义、作用、场景、目的及关联限制 | 填写要求 | 填写格式与内容 | 默认和省略行为 | 修改生效及影响 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| compose.fnos.yaml | x-https-port | __FRONTEND_PORT__:__FRONTEND_PORT__ | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-ca-port | __CA_PORT__:__CA_PORT__ | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_HOSTS |  | 证书访问地址，prepare 脚本使用空格分隔，包含真实客户端访问主机 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_BASE_PATH |  | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_DOMAIN |  | 证书附加域名，不代表自动签发公开可信证书 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_FRONTEND_PORT | __FRONTEND_PORT__ | 共享表前端端口，模板生成时替换为发行端口 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_BACKEND_PORT | __BACKEND_PORT__ | 共享表业务端口，映射和健康检查须一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_CA_PORT | __CA_PORT__ | 准备流程公开 CA 入口端口，需与发行表对应 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_PUBLIC_DIR | /public-ca | prepare 输出公开证书目录，与公开证书卷对应 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | x-settings.LEAFDOC_MANAGED_GATEWAY | true | 启用受管理准备模式，参数变化或证书临近过期时检查更新 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.image | __APP_IMAGE__ | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.pull_policy | never | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.user | 0 | 容器执行身份，prepare 用 0 获取卷权限，业务保持镜像默认非 root 身份 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.entrypoint | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.entrypoint[0] | sh | 覆盖容器入口，prepare 通过 sh 运行实例准备脚本 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.entrypoint[1] | /prepare-instance.sh | 覆盖容器入口，prepare 通过 sh 运行实例准备脚本 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_HOSTS |  | 证书访问地址，prepare 脚本使用空格分隔，包含真实客户端访问主机 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_BASE_PATH |  | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_DOMAIN |  | 证书附加域名，不代表自动签发公开可信证书 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_FRONTEND_PORT | __FRONTEND_PORT__ | 共享表前端端口，模板生成时替换为发行端口 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_BACKEND_PORT | __BACKEND_PORT__ | 共享表业务端口，映射和健康检查须一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_CA_PORT | __CA_PORT__ | 准备流程公开 CA 入口端口，需与发行表对应 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_PUBLIC_DIR | /public-ca | prepare 输出公开证书目录，与公开证书卷对应 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.environment.LEAFDOC_MANAGED_GATEWAY | true | 启用受管理准备模式，参数变化或证书临近过期时检查更新 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.volumes[0] | instance_data:/instance-data | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.volumes[1] | instance_config:/config | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.volumes[2] | public_certificates:/public-ca | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.prepare.restart | no | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.image | __APP_IMAGE__ | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.pull_policy | never | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.restart | unless-stopped | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.depends_on | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.depends_on.prepare | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.depends_on.prepare.condition | service_completed_successfully | 等待 prepare 一次性任务成功结束，不要求它持续运行 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_HOSTS |  | 证书访问地址，prepare 脚本使用空格分隔，包含真实客户端访问主机 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_BASE_PATH |  | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_DOMAIN |  | 证书附加域名，不代表自动签发公开可信证书 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_FRONTEND_PORT | __FRONTEND_PORT__ | 共享表前端端口，模板生成时替换为发行端口 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_BACKEND_PORT | __BACKEND_PORT__ | 共享表业务端口，映射和健康检查须一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_CA_PORT | __CA_PORT__ | 准备流程公开 CA 入口端口，需与发行表对应 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_PUBLIC_DIR | /public-ca | prepare 输出公开证书目录，与公开证书卷对应 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_MANAGED_GATEWAY | true | 启用受管理准备模式，参数变化或证书临近过期时检查更新 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_HOST | 0.0.0.0 | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_TLS_CERT | /run/leafdoc-config/tls/server.crt | 容器内服务证书路径，须与只读 TLS 挂载一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_TLS_KEY | /run/leafdoc-config/tls/server.key | 容器内证书私钥路径，需可读并与证书匹配 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_WEB_SETUP | true | 开启未安装实例的网页安装入口，已安装状态仍按持久记录判断 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_DATA_DIR | /app/data | 容器内实例文件位置，必须与持久挂载对应，不是 MySQL 数据目录 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.environment.LEAFDOC_SETUP_CA_FILE | /run/leafdoc-public/local-ca.crt | 网页安装流程提供的公开 CA 文件，不能指向私钥 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.ports | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.ports[0] | __BACKEND_PORT__:__BACKEND_PORT__ | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.command | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.command[0] | --enable-revision | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.command[1] | --snapshot | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.volumes[0] | instance_config:/run/leafdoc-config:ro | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.volumes[1] | instance_data:/app/data | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.app.volumes[2] | public_certificates:/run/leafdoc-public:ro | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.image | __APP_IMAGE__ | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.pull_policy | never | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.restart | unless-stopped | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.command | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.command[0] | gateway | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.environment.LEAFDOC_HOST | 0.0.0.0 | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.environment.LEAFDOC_BACKEND_URL | http://app:__BACKEND_PORT__ | 网关转发目标，服务名 app 只在对应 Compose 网络内解析 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.environment.LEAFDOC_TLS_CERT | /run/leafdoc-config/tls/server.crt | 容器内服务证书路径，须与只读 TLS 挂载一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.environment.LEAFDOC_TLS_KEY | /run/leafdoc-config/tls/server.key | 容器内证书私钥路径，需可读并与证书匹配 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.depends_on | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.depends_on.prepare | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.depends_on.prepare.condition | service_completed_successfully | 等待 prepare 一次性任务成功结束，不要求它持续运行 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.ports | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.ports[0] | __FRONTEND_PORT__:__FRONTEND_PORT__ | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | services.gateway.volumes[0] | instance_config:/run/leafdoc-config:ro | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | volumes | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | volumes.instance_data | null | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 空命名卷声明，Compose 创建或复用项目范围命名卷 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | volumes.instance_config | null | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 空命名卷声明，Compose 创建或复用项目范围命名卷 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.fnos.yaml | volumes.public_certificates | null | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 空命名卷声明，Compose 创建或复用项目范围命名卷 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |

### 必须填写

| 完整路径 | 要求与原因 |
| --- | --- |
| 无 | 本文件没有此类字段 |

### 不需要填写

| 完整路径 | 要求与原因 |
| --- | --- |
| x-https-port | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| x-ca-port | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| x-settings | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| x-settings.LEAFDOC_FRONTEND_PORT | 共享表前端端口，模板生成时替换为发行端口 |
| x-settings.LEAFDOC_BACKEND_PORT | 共享表业务端口，映射和健康检查须一致 |
| x-settings.LEAFDOC_CA_PORT | 准备流程公开 CA 入口端口，需与发行表对应 |
| x-settings.LEAFDOC_PUBLIC_DIR | prepare 输出公开证书目录，与公开证书卷对应 |
| x-settings.LEAFDOC_MANAGED_GATEWAY | 启用受管理准备模式，参数变化或证书临近过期时检查更新 |
| services | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.prepare | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.prepare.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |
| services.prepare.pull_policy | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 |
| services.prepare.user | 容器执行身份，prepare 用 0 获取卷权限，业务保持镜像默认非 root 身份 |
| services.prepare.entrypoint | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.prepare.entrypoint[0] | 覆盖容器入口，prepare 通过 sh 运行实例准备脚本 |
| services.prepare.entrypoint[1] | 覆盖容器入口，prepare 通过 sh 运行实例准备脚本 |
| services.prepare.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.prepare.environment.LEAFDOC_FRONTEND_PORT | 共享表前端端口，模板生成时替换为发行端口 |
| services.prepare.environment.LEAFDOC_BACKEND_PORT | 共享表业务端口，映射和健康检查须一致 |
| services.prepare.environment.LEAFDOC_CA_PORT | 准备流程公开 CA 入口端口，需与发行表对应 |
| services.prepare.environment.LEAFDOC_PUBLIC_DIR | prepare 输出公开证书目录，与公开证书卷对应 |
| services.prepare.environment.LEAFDOC_MANAGED_GATEWAY | 启用受管理准备模式，参数变化或证书临近过期时检查更新 |
| services.prepare.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.prepare.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.prepare.volumes[1] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.prepare.volumes[2] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.prepare.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.app | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |
| services.app.pull_policy | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 |
| services.app.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.app.depends_on | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.depends_on.prepare | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.depends_on.prepare.condition | 等待 prepare 一次性任务成功结束，不要求它持续运行 |
| services.app.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.environment.LEAFDOC_FRONTEND_PORT | 共享表前端端口，模板生成时替换为发行端口 |
| services.app.environment.LEAFDOC_BACKEND_PORT | 共享表业务端口，映射和健康检查须一致 |
| services.app.environment.LEAFDOC_CA_PORT | 准备流程公开 CA 入口端口，需与发行表对应 |
| services.app.environment.LEAFDOC_PUBLIC_DIR | prepare 输出公开证书目录，与公开证书卷对应 |
| services.app.environment.LEAFDOC_MANAGED_GATEWAY | 启用受管理准备模式，参数变化或证书临近过期时检查更新 |
| services.app.environment.LEAFDOC_HOST | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 |
| services.app.environment.LEAFDOC_TLS_CERT | 容器内服务证书路径，须与只读 TLS 挂载一致 |
| services.app.environment.LEAFDOC_TLS_KEY | 容器内证书私钥路径，需可读并与证书匹配 |
| services.app.environment.LEAFDOC_WEB_SETUP | 开启未安装实例的网页安装入口，已安装状态仍按持久记录判断 |
| services.app.environment.LEAFDOC_DATA_DIR | 容器内实例文件位置，必须与持久挂载对应，不是 MySQL 数据目录 |
| services.app.environment.LEAFDOC_SETUP_CA_FILE | 网页安装流程提供的公开 CA 文件，不能指向私钥 |
| services.app.ports | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.ports[0] | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| services.app.command | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.command[0] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.app.command[1] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.app.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.app.volumes[1] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.app.volumes[2] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.gateway | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |
| services.gateway.pull_policy | never 禁止拉取，镜像未加载会失败，防止离线运行误访问镜像仓库 |
| services.gateway.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.gateway.command | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.command[0] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.gateway.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.environment.LEAFDOC_HOST | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 |
| services.gateway.environment.LEAFDOC_BACKEND_URL | 网关转发目标，服务名 app 只在对应 Compose 网络内解析 |
| services.gateway.environment.LEAFDOC_TLS_CERT | 容器内服务证书路径，须与只读 TLS 挂载一致 |
| services.gateway.environment.LEAFDOC_TLS_KEY | 容器内证书私钥路径，需可读并与证书匹配 |
| services.gateway.depends_on | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.depends_on.prepare | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.depends_on.prepare.condition | 等待 prepare 一次性任务成功结束，不要求它持续运行 |
| services.gateway.ports | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.ports[0] | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| services.gateway.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| volumes | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| volumes.instance_data | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 |
| volumes.instance_config | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 |
| volumes.public_certificates | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 |

### 可以填写也可以不填写

| 完整路径 | 要求与原因 |
| --- | --- |
| x-settings.LEAFDOC_HOSTS | 证书访问地址，prepare 脚本使用空格分隔，包含真实客户端访问主机 |
| x-settings.LEAFDOC_BASE_PATH | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 |
| x-settings.LEAFDOC_DOMAIN | 证书附加域名，不代表自动签发公开可信证书 |
| services.prepare.environment.LEAFDOC_HOSTS | 证书访问地址，prepare 脚本使用空格分隔，包含真实客户端访问主机 |
| services.prepare.environment.LEAFDOC_BASE_PATH | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 |
| services.prepare.environment.LEAFDOC_DOMAIN | 证书附加域名，不代表自动签发公开可信证书 |
| services.app.environment.LEAFDOC_HOSTS | 证书访问地址，prepare 脚本使用空格分隔，包含真实客户端访问主机 |
| services.app.environment.LEAFDOC_BASE_PATH | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 |
| services.app.environment.LEAFDOC_DOMAIN | 证书附加域名，不代表自动签发公开可信证书 |

## other/compose.public.yaml

| 所属文件 | 完整路径 | 当前值 | 含义、作用、场景、目的及关联限制 | 填写要求 | 填写格式与内容 | 默认和省略行为 | 修改生效及影响 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| compose.public.yaml | services | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.public.yaml | services.gateway | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.public.yaml | services.gateway.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.public.yaml | services.gateway.environment.LEAFDOC_HOST | 0.0.0.0 | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |

### 必须填写

| 完整路径 | 要求与原因 |
| --- | --- |
| 无 | 本文件没有此类字段 |

### 不需要填写

| 完整路径 | 要求与原因 |
| --- | --- |
| services | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.environment.LEAFDOC_HOST | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 |

### 可以填写也可以不填写

| 完整路径 | 要求与原因 |
| --- | --- |
| 无 | 本文件没有此类字段 |

## other/compose.yaml

| 所属文件 | 完整路径 | 当前值 | 含义、作用、场景、目的及关联限制 | 填写要求 | 填写格式与内容 | 默认和省略行为 | 修改生效及影响 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| compose.yaml | services | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.image | ${LEAFDOC_MYSQL_IMAGE:?Run deploy.sh init first} | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_IMAGE 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.restart | unless-stopped | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.environment.MYSQL_ROOT_PASSWORD | ${LEAFDOC_MYSQL_ROOT_PASSWORD:?} | MySQL 镜像初始化 root 密码，初始化已有数据库卷时不用于重置密码 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_ROOT_PASSWORD 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.environment.MYSQL_DATABASE | ${LEAFDOC_MYSQL_DATABASE:?} | MySQL 镜像首次初始化创建的库，已有卷不重复初始化 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_DATABASE 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.environment.MYSQL_USER | ${LEAFDOC_MYSQL_USER:?} | MySQL 首次初始化创建的应用用户，已有卷不自动修改 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_USER 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.environment.MYSQL_PASSWORD | ${LEAFDOC_MYSQL_PASSWORD:?} | MySQL 首次初始化应用密码，需与应用连接密码一致 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_PASSWORD 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.volumes[0] | database:/var/lib/mysql | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck.test | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck.test[0] | CMD-SHELL | 在容器 shell 中执行探测，支持环境变量展开 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck.test[1] | MYSQL_PWD="$$MYSQL_PASSWORD" mysql --protocol=tcp -h 127.0.0.1 -u "$$MYSQL_USER" "$$MYSQL_DATABASE" -e "SELECT 1" > /dev/null | 健康探测命令项，保持列表顺序；数据库检查执行 SELECT 1 验证连接和凭据 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck.interval | 10s | 健康探测间隔，持续时间格式，影响检查频率 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck.timeout | 5s | 单次健康探测超时，超过时本轮失败 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck.retries | 30 | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 | 不需要填写 | 整数；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.healthcheck.start_period | 60s | 启动宽限时间，避免初始化阶段过早计入失败 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.logging | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.logging.driver | json-file | 日志驱动 json-file，Docker 管理标准输出和错误输出日志 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.logging.options | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.logging.options.max-size | 10m | 单个容器日志文件轮转大小，不影响业务数据 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.database.logging.options.max-file | 3 | 日志轮转保留文件数，字符串值传给日志驱动 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.image | ${LEAFDOC_IMAGE:-leafdoc:local} | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_IMAGE 缺失时使用 leafdoc:local | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.restart | unless-stopped | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.depends_on | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.depends_on.database | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.depends_on.database.condition | service_healthy | 等待被依赖服务健康后启动，依赖必须声明健康探测 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_HOST | 0.0.0.0 | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_TLS_CERT | /run/leafdoc-tls/server.crt | 容器内服务证书路径，须与只读 TLS 挂载一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_TLS_KEY | /run/leafdoc-tls/server.key | 容器内证书私钥路径，需可读并与证书匹配 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_MYSQL_HOST | database | MySQL 可达主机，源码栈用 database 服务名，不能用容器 localhost 指代宿主 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_MYSQL_DATABASE | ${LEAFDOC_MYSQL_DATABASE:?} | 业务数据库名称，需满足连接用户权限和安装结构检查 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_DATABASE 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_MYSQL_USER | ${LEAFDOC_MYSQL_USER:?} | 数据库连接账户，需具备实际业务及初始化所需权限 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_USER 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_MYSQL_PASSWORD | ${LEAFDOC_MYSQL_PASSWORD:?} | 数据库账户密码，与数据库端设置一致，不写入版本库 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_MYSQL_PASSWORD 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_JWT_SECRET | ${LEAFDOC_JWT_SECRET:?} | 会话签名密钥，保持私密并持久保存，改变会影响已有会话 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_JWT_SECRET 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_ADMIN_PASSWORD | ${LEAFDOC_ADMIN_PASSWORD:?} | 初始化管理员密码，不能用它重置已存在账户 | 必须填写 | 字符串或标量，保持当前 YAML 类型；由部署脚本生成或传入，禁止提交私密值 | LEAFDOC_ADMIN_PASSWORD 缺失时拒绝解析 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_BASE_PATH | ${LEAFDOC_BASE_PATH:-} | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BASE_PATH 缺失时使用 空字符串 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.environment.LEAFDOC_DATA_DIR | /app/data | 容器内实例文件位置，必须与持久挂载对应，不是 MySQL 数据目录 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.ports | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.ports[0] | ${LEAFDOC_BIND_ADDRESS:-0.0.0.0}:${LEAFDOC_BACKEND_PORT}:${LEAFDOC_BACKEND_PORT}/tcp | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BIND_ADDRESS 缺失时使用 0.0.0.0；LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.command | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.command[0] | --enable-revision | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.command[1] | --snapshot | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.volumes[0] | ${LEAFDOC_CONFIG_DIR:-/etc/leafdoc}/tls:/run/leafdoc-tls:ro | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_CONFIG_DIR 缺失时使用 /etc/leafdoc | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.volumes[1] | documents:/app/data | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck.test | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck.test[0] | CMD-SHELL | 在容器 shell 中执行探测，支持环境变量展开 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck.test[1] | wget -qO- "http://127.0.0.1:${LEAFDOC_BACKEND_PORT}$${LEAFDOC_BASE_PATH}/api/health" > /dev/null | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_BASE_PATH 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck.interval | 10s | 健康探测间隔，持续时间格式，影响检查频率 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck.timeout | 5s | 单次健康探测超时，超过时本轮失败 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck.retries | 12 | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 | 不需要填写 | 整数；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.healthcheck.start_period | 30s | 启动宽限时间，避免初始化阶段过早计入失败 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.stop_grace_period | 45s | 优雅停止等待时间，超时后 Docker 强制结束进程 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.logging | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.logging.driver | json-file | 日志驱动 json-file，Docker 管理标准输出和错误输出日志 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.logging.options | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.logging.options.max-size | 10m | 单个容器日志文件轮转大小，不影响业务数据 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.app.logging.options.max-file | 3 | 日志轮转保留文件数，字符串值传给日志驱动 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.image | ${LEAFDOC_IMAGE:-leafdoc:local} | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_IMAGE 缺失时使用 leafdoc:local | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.restart | unless-stopped | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.command | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.command[0] | gateway | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.environment | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.environment.LEAFDOC_HOST | 0.0.0.0 | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.environment.LEAFDOC_BASE_PATH | ${LEAFDOC_BASE_PATH:-} | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BASE_PATH 缺失时使用 空字符串 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.environment.LEAFDOC_BACKEND_URL | http://app:${LEAFDOC_BACKEND_PORT} | 网关转发目标，服务名 app 只在对应 Compose 网络内解析 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BACKEND_PORT 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.environment.LEAFDOC_TLS_CERT | /run/leafdoc-tls/server.crt | 容器内服务证书路径，须与只读 TLS 挂载一致 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.environment.LEAFDOC_TLS_KEY | /run/leafdoc-tls/server.key | 容器内证书私钥路径，需可读并与证书匹配 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.depends_on | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.depends_on[0] | app | 依赖服务名或条件，不是容器外部数据库地址；简单列表不等待健康成功 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.ports | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.ports[0] | ${LEAFDOC_BIND_ADDRESS:-0.0.0.0}:${LEAFDOC_FRONTEND_PORT}:${LEAFDOC_FRONTEND_PORT}/tcp | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_BIND_ADDRESS 缺失时使用 0.0.0.0；LEAFDOC_FRONTEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_FRONTEND_PORT 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.volumes | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.volumes[0] | ${LEAFDOC_CONFIG_DIR:-/etc/leafdoc}/tls:/run/leafdoc-tls:ro | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 | 可以填写也可以不填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_CONFIG_DIR 缺失时使用 /etc/leafdoc | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.healthcheck | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.healthcheck.test | [列表] | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 | 不需要填写 | 列表；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.healthcheck.test[0] | CMD-SHELL | 在容器 shell 中执行探测，支持环境变量展开 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.healthcheck.test[1] | wget -qO- "http://127.0.0.1:${LEAFDOC_FRONTEND_PORT}$${LEAFDOC_BASE_PATH}/api/health" > /dev/null | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | LEAFDOC_FRONTEND_PORT 无文件内替代默认，须由部署入口提供；LEAFDOC_BASE_PATH 无文件内替代默认，须由部署入口提供 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.healthcheck.interval | 10s | 健康探测间隔，持续时间格式，影响检查频率 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.healthcheck.timeout | 5s | 单次健康探测超时，超过时本轮失败 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.healthcheck.retries | 12 | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 | 不需要填写 | 整数；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | services.gateway.stop_grace_period | 30s | 优雅停止等待时间，超时后 Docker 强制结束进程 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | volumes | {映射} | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 | 不需要填写 | 映射；固定项保留原值，自定义项按本行用途设置 | 文件显式值，无项目级省略默认；不要删除必需结构 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | volumes.database | null | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 空命名卷声明，Compose 创建或复用项目范围命名卷 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |
| compose.yaml | volumes.documents | null | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 | 不需要填写 | 字符串或标量，保持当前 YAML 类型；固定项保留原值，自定义项按本行用途设置 | 空命名卷声明，Compose 创建或复用项目范围命名卷 | Compose 解析并重新创建受影响容器；已有卷和数据库不因环境变化自动重置 |

### 必须填写

| 完整路径 | 要求与原因 |
| --- | --- |
| services.database.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |
| services.database.environment.MYSQL_ROOT_PASSWORD | MySQL 镜像初始化 root 密码，初始化已有数据库卷时不用于重置密码 |
| services.database.environment.MYSQL_DATABASE | MySQL 镜像首次初始化创建的库，已有卷不重复初始化 |
| services.database.environment.MYSQL_USER | MySQL 首次初始化创建的应用用户，已有卷不自动修改 |
| services.database.environment.MYSQL_PASSWORD | MySQL 首次初始化应用密码，需与应用连接密码一致 |
| services.app.environment.LEAFDOC_MYSQL_DATABASE | 业务数据库名称，需满足连接用户权限和安装结构检查 |
| services.app.environment.LEAFDOC_MYSQL_USER | 数据库连接账户，需具备实际业务及初始化所需权限 |
| services.app.environment.LEAFDOC_MYSQL_PASSWORD | 数据库账户密码，与数据库端设置一致，不写入版本库 |
| services.app.environment.LEAFDOC_JWT_SECRET | 会话签名密钥，保持私密并持久保存，改变会影响已有会话 |
| services.app.environment.LEAFDOC_ADMIN_PASSWORD | 初始化管理员密码，不能用它重置已存在账户 |

### 不需要填写

| 完整路径 | 要求与原因 |
| --- | --- |
| services | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.database | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.database.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.database.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.database.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.database.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.database.healthcheck | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.database.healthcheck.test | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.database.healthcheck.test[0] | 在容器 shell 中执行探测，支持环境变量展开 |
| services.database.healthcheck.test[1] | 健康探测命令项，保持列表顺序；数据库检查执行 SELECT 1 验证连接和凭据 |
| services.database.healthcheck.interval | 健康探测间隔，持续时间格式，影响检查频率 |
| services.database.healthcheck.timeout | 单次健康探测超时，超过时本轮失败 |
| services.database.healthcheck.retries | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 |
| services.database.healthcheck.start_period | 启动宽限时间，避免初始化阶段过早计入失败 |
| services.database.logging | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.database.logging.driver | 日志驱动 json-file，Docker 管理标准输出和错误输出日志 |
| services.database.logging.options | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.database.logging.options.max-size | 单个容器日志文件轮转大小，不影响业务数据 |
| services.database.logging.options.max-file | 日志轮转保留文件数，字符串值传给日志驱动 |
| services.app | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.app.depends_on | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.depends_on.database | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.depends_on.database.condition | 等待被依赖服务健康后启动，依赖必须声明健康探测 |
| services.app.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.environment.LEAFDOC_HOST | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 |
| services.app.environment.LEAFDOC_TLS_CERT | 容器内服务证书路径，须与只读 TLS 挂载一致 |
| services.app.environment.LEAFDOC_TLS_KEY | 容器内证书私钥路径，需可读并与证书匹配 |
| services.app.environment.LEAFDOC_MYSQL_HOST | MySQL 可达主机，源码栈用 database 服务名，不能用容器 localhost 指代宿主 |
| services.app.environment.LEAFDOC_DATA_DIR | 容器内实例文件位置，必须与持久挂载对应，不是 MySQL 数据目录 |
| services.app.ports | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.command | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.command[0] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.app.command[1] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.app.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.volumes[1] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.app.healthcheck | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.healthcheck.test | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.app.healthcheck.test[0] | 在容器 shell 中执行探测，支持环境变量展开 |
| services.app.healthcheck.test[1] | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 |
| services.app.healthcheck.interval | 健康探测间隔，持续时间格式，影响检查频率 |
| services.app.healthcheck.timeout | 单次健康探测超时，超过时本轮失败 |
| services.app.healthcheck.retries | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 |
| services.app.healthcheck.start_period | 启动宽限时间，避免初始化阶段过早计入失败 |
| services.app.stop_grace_period | 优雅停止等待时间，超时后 Docker 强制结束进程 |
| services.app.logging | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.logging.driver | 日志驱动 json-file，Docker 管理标准输出和错误输出日志 |
| services.app.logging.options | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.app.logging.options.max-size | 单个容器日志文件轮转大小，不影响业务数据 |
| services.app.logging.options.max-file | 日志轮转保留文件数，字符串值传给日志驱动 |
| services.gateway | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.restart | unless-stopped 在非人工停止后恢复，no 不自动重启；不等同健康失败自动重启 |
| services.gateway.command | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.command[0] | 覆盖镜像默认参数，gateway 启动网关，enable-revision 和 snapshot 开启对应能力 |
| services.gateway.environment | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.environment.LEAFDOC_HOST | 业务监听地址，0.0.0.0 接受容器网络请求，回环地址仅本容器可达 |
| services.gateway.environment.LEAFDOC_BACKEND_URL | 网关转发目标，服务名 app 只在对应 Compose 网络内解析 |
| services.gateway.environment.LEAFDOC_TLS_CERT | 容器内服务证书路径，须与只读 TLS 挂载一致 |
| services.gateway.environment.LEAFDOC_TLS_KEY | 容器内证书私钥路径，需可读并与证书匹配 |
| services.gateway.depends_on | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.depends_on[0] | 依赖服务名或条件，不是容器外部数据库地址；简单列表不等待健康成功 |
| services.gateway.ports | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.volumes | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.healthcheck | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| services.gateway.healthcheck.test | 有序列表，逐项含义见各下标行，不能把列表写成逗号拼接字符串 |
| services.gateway.healthcheck.test[0] | 在容器 shell 中执行探测，支持环境变量展开 |
| services.gateway.healthcheck.test[1] | 从容器内部请求本服务健康接口，端口和路径前缀必须匹配，失败标记不健康 |
| services.gateway.healthcheck.interval | 健康探测间隔，持续时间格式，影响检查频率 |
| services.gateway.healthcheck.timeout | 单次健康探测超时，超过时本轮失败 |
| services.gateway.healthcheck.retries | 连续探测失败阈值，达到后标记 unhealthy，不直接重启容器 |
| services.gateway.stop_grace_period | 优雅停止等待时间，超时后 Docker 强制结束进程 |
| volumes | 映射结构，收拢下表所列子字段；空映射由所属工具按此对象声明处理 |
| volumes.database | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 |
| volumes.documents | 命名卷或模板结构声明，含义由所在父字段及对应子项共同确定，不作为业务环境变量读取 |

### 可以填写也可以不填写

| 完整路径 | 要求与原因 |
| --- | --- |
| services.app.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |
| services.app.environment.LEAFDOC_BASE_PATH | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 |
| services.app.ports[0] | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| services.app.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |
| services.gateway.image | 镜像引用，离线包要求已 load；模板占位符在发行时替换，变量必须可解析 |
| services.gateway.environment.LEAFDOC_BASE_PATH | URL 路径前缀，为空使用根路径，非空以斜线开头并与访问 URL 一致 |
| services.gateway.ports[0] | 宿主端口到容器端口映射；绑定地址控制可访问网卡，发行占位符由 package.mjs 替换 |
| services.gateway.volumes[0] | 卷或绑定挂载，冒号左侧为来源、右侧为容器目录，ro 表示只读；错误映射导致配置不可读或数据不持久 |

## 发行生成文件

| 文件 | 对应关系 | 填写与生效 |
| --- | --- | --- |
| fnos/docker-compose.yml | 由 other/compose.fnos.yaml 替换 APP_IMAGE、GATEWAY_IMAGE、FRONTEND_PORT、BACKEND_PORT、CA_PORT 后生成，字段与上述表逐一对应 | 地址和可选前缀按目标机设置，其他替换值由发行脚本提供 |
| 包内 compose.yaml | 直接复制 other/compose.bundle.yaml | 使用包内 install.sh 解析部署环境 |
| Windows .runtime/compose.yaml | start.ps1 动态生成 JSON 形式的合法 YAML | 运行目录与实例路径动态确定，不提交到仓库 |

## 命令行与环境变量来源

| 命令行选项 | 环境变量 | 代码声明默认值 | 唯一声明来源 |
| --- | --- | --- | --- |
| `jwt-secret` | `LEAFDOC_JWT_SECRET` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_auth.go |
| `public-access` | `LEAFDOC_PUBLIC_ACCESS` | `defaults.PublicAccess` | backend/cmd/leafdoc/flags_auth.go |
| `access-token-timeout` | `LEAFDOC_ACCESS_TOKEN_TIMEOUT` | `15 * time.Minute` | backend/cmd/leafdoc/flags_auth.go |
| `refresh-token-timeout` | `LEAFDOC_REFRESH_TOKEN_TIMEOUT` | `7 * 24 * time.Hour` | backend/cmd/leafdoc/flags_auth.go |
| `admin-username` | `LEAFDOC_ADMIN_USERNAME` | `defaults.AdminUsername` | backend/cmd/leafdoc/flags_auth.go |
| `admin-email` | `LEAFDOC_ADMIN_EMAIL` | `defaults.AdminEmail` | backend/cmd/leafdoc/flags_auth.go |
| `admin-password` | `LEAFDOC_ADMIN_PASSWORD` | `defaults.AdminPassword` | backend/cmd/leafdoc/flags_auth.go |
| `editor-limit` | `LEAFDOC_EDITOR_LIMIT` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_auth.go |
| `snapshot` | `LEAFDOC_SNAPSHOT` | `true` | backend/cmd/leafdoc/flags_backup.go |
| `snapshot-interval` | `LEAFDOC_SNAPSHOT_INTERVAL` | `24 * time.Hour` | backend/cmd/leafdoc/flags_backup.go |
| `snapshot-retention` | `LEAFDOC_SNAPSHOT_RETENTION` | `10` | backend/cmd/leafdoc/flags_backup.go |
| `snapshot-dir` | `LEAFDOC_SNAPSHOT_DIR` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_backup.go |
| `restore-upload-max-size` | `LEAFDOC_RESTORE_UPLOAD_MAX_SIZE` | `"500MiB"` | backend/cmd/leafdoc/flags_backup.go |
| `smtp-host` | `LEAFDOC_SMTP_HOST` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_email.go |
| `smtp-port` | `LEAFDOC_SMTP_PORT` | `587` | backend/cmd/leafdoc/flags_email.go |
| `smtp-username` | `LEAFDOC_SMTP_USERNAME` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_email.go |
| `smtp-password` | `LEAFDOC_SMTP_PASSWORD` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_email.go |
| `smtp-from` | `LEAFDOC_SMTP_FROM` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_email.go |
| `smtp-from-name` | `LEAFDOC_SMTP_FROM_NAME` | `"LeafDoc"` | backend/cmd/leafdoc/flags_email.go |
| `smtp-security` | `LEAFDOC_SMTP_SECURITY` | `"starttls"` | backend/cmd/leafdoc/flags_email.go |
| `smtp-insecure-skip-verify` | `LEAFDOC_SMTP_INSECURE_SKIP_VERIFY` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_email.go |
| `smtp-timeout` | `LEAFDOC_SMTP_TIMEOUT` | `10 * time.Second` | backend/cmd/leafdoc/flags_email.go |
| `public-url` | `LEAFDOC_PUBLIC_URL` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_email.go |
| `default-language` | `LEAFDOC_DEFAULT_LANGUAGE` | `defaults.Language` | backend/cmd/leafdoc/flags_frontend.go |
| `inject-code-in-header` | `LEAFDOC_INJECT_CODE_IN_HEADER` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_frontend.go |
| `custom-stylesheet` | `LEAFDOC_CUSTOM_STYLESHEET` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_frontend.go |
| `hide-link-metadata-section` | `LEAFDOC_HIDE_LINK_METADATA_SECTION` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_frontend.go |
| `max-asset-upload-size` | `LEAFDOC_MAX_ASSET_UPLOAD_SIZE` | `"50MiB"` | backend/cmd/leafdoc/flags_frontend.go |
| `enable-revision` | `LEAFDOC_ENABLE_REVISION` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_frontend.go |
| `enable-link-refactor` | `LEAFDOC_ENABLE_LINK_REFACTOR` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_frontend.go |
| `enable-api-key-management` | `LEAFDOC_ENABLE_API_KEY_MANAGEMENT` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_frontend.go |
| `max-revision-history` | `LEAFDOC_MAX_REVISION_HISTORY` | `100` | backend/cmd/leafdoc/flags_frontend.go |
| `revision-coalesce-window` | `LEAFDOC_REVISION_COALESCE_WINDOW` | `5 * time.Minute` | backend/cmd/leafdoc/flags_frontend.go |
| `enable-metrics` | `LEAFDOC_ENABLE_METRICS` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_metrics.go |
| `metrics-host` | `LEAFDOC_METRICS_HOST` | `"127.0.0.1"` | backend/cmd/leafdoc/flags_metrics.go |
| `metrics-port` | `LEAFDOC_METRICS_PORT` | `serviceports.Port("metrics")` | backend/cmd/leafdoc/flags_metrics.go |
| `enable-http-remote-user` | `LEAFDOC_ENABLE_HTTP_REMOTE_USER` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `http-remote-user-header-name` | `LEAFDOC_HTTP_REMOTE_USER_HEADER_NAME` | `"Remote-User"` | backend/cmd/leafdoc/flags_proxy.go |
| `enable-http-remote-user-auto-create` | `LEAFDOC_ENABLE_HTTP_REMOTE_USER_AUTO_CREATE` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `http-remote-user-email-header-name` | `LEAFDOC_HTTP_REMOTE_USER_EMAIL_HEADER_NAME` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `http-remote-user-default-role` | `LEAFDOC_HTTP_REMOTE_USER_DEFAULT_ROLE` | `"viewer"` | backend/cmd/leafdoc/flags_proxy.go |
| `trusted-proxy-ips` | `LEAFDOC_TRUSTED_PROXY_IPS` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `http-remote-user-logout-url` | `LEAFDOC_HTTP_REMOTE_USER_LOGOUT_URL` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `login-url` | `LEAFDOC_LOGIN_URL` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `logout-url` | `LEAFDOC_LOGOUT_URL` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `user-management-url` | `LEAFDOC_USER_MANAGEMENT_URL` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_proxy.go |
| `host` | `LEAFDOC_HOST` | `"127.0.0.1"` | backend/cmd/leafdoc/flags_server.go |
| `port` | `LEAFDOC_PORT` | `serviceports.Port("backend")` | backend/cmd/leafdoc/flags_server.go |
| `unix-socket` | `LEAFDOC_UNIX_SOCKET` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_server.go |
| `data-dir` | `LEAFDOC_DATA_DIR` | `"./data"` | backend/cmd/leafdoc/flags_server.go |
| `base-path` | `LEAFDOC_BASE_PATH` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_server.go |
| `allow-insecure` | `LEAFDOC_ALLOW_INSECURE` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_server.go |
| `log-format` | `LEAFDOC_LOG_FORMAT` | `"text"` | backend/cmd/leafdoc/flags_server.go |
| `disable-request-log` | `LEAFDOC_DISABLE_REQUEST_LOG` | `未显式声明，按选项类型零值及启动校验处理` | backend/cmd/leafdoc/flags_server.go |

| 使用规则 | 说明 |
| --- | --- |
| 参数来源 | CLI 选项绑定对应环境变量，安装状态和部署入口还会提供持久值，不用表中零值推断最终已安装连接参数 |
| 证书 | 路径必须对应实际挂载，私钥与证书成对 |
| 邮件 | 邀请和密码恢复需要可用 SMTP，未配置不宣称邮件已发送 |
| 版本与快照 | 按对应开关启用，功能未启用时按接口实际响应处理 |
| 固定端口 | serviceports.Port 的值来自共享 ports.json，校验不接受任意端口 |
