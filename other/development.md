# 开发、检查与维护

## 模块与修改入口

前端 app 装配应用，features 按功能组织界面，shared 提供 API、组件和规则。后端 cmd 负责参数与启动，wiki 负责路由和用例，core 负责树、存储、认证和其他基础能力。业务写入通过页面保存编排与工作单元，不能在新路由绕过版本和权限校验直接改数据库。

## Ubuntu 开发环境

宿主需要 Git、Node.js 与 Linux Docker。编译依赖按 Dockerfile.tools 和锁文件安装，避免宿主工具版本替代容器版本。从项目根目录执行：

```sh
node other/scripts/container.mjs verify
node other/scripts/container.mjs test
node other/scripts/container.mjs vet
node other/scripts/container.mjs lint
node other/scripts/container.mjs ui-check
node other/scripts/container.mjs schema
```

verify 检查结构、私有引用、工具版本与数据库文件。test 运行后端测试，vet 做 Go 静态分析，lint 使用固定版本检查器，ui-check 包含前端测试、lint、格式、结构和构建。schema 对比数据库默认结构，只有有意改变默认结构时才使用 schema-write 生成文件。

```sh
node other/scripts/container.mjs build
sudo bash other/scripts/deploy.sh init --hosts localhost,127.0.0.1
sudo bash other/scripts/deploy.sh start
```

build 与 release 都调用 package.mjs，生成完整发行包，不会推送远端镜像。Ubuntu 启停见部署文档。Windows 本地入口是 PowerShell 7 执行 other/scripts/start.ps1，带 Build 重建镜像，Stop 停止当前实例。

## 默认教程

backend/internal/core/defaultdocs 的清单集中维护标题、slug 与 ParentSlug，正文通过 go:embed 嵌入。父节点必须先于子节点，根节点作为带正文的 section，子节点作为 page。完整路径由父级计算，正文链接与真实路径一致。

首次初始化由 wiki/default_documents.go 执行，数据库 instance_state 中记录完成状态。已完成实例不会自动补写或重置教程，不能靠删除源码正文清空已有数据库页面。

修改清单后同步正文与链接测试，验证数量、层级、恢复初始化和站内目标。导入 fixtures 是真实测试输入，改正文时检查相对链接、标题、资源和测试断言，不删除二进制资源。

## 维护纪律

运行配置、凭据、缓存、归档和本机验证资料不进入提交。提交前核对忽略规则，不修改 AGENTS.md 和策略钩子。不强制推送，不重写远端历史。日志中的错误按真实根因处理，不降低日志级别掩盖。

接口变更同步 API 文档，运行参数变更同步配置表，发行步骤变更同步包内 README 来源。更改后执行受影响测试及项目检查，再通过真实启动入口验证访问和登录。
