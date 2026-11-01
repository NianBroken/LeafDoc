# fnOS Docker 纯可视化部署与 Compose 配置维护说明

本文是 LeafDoc fnOS 部署资料的维护文档。适用对象为 x64 fnOS，验收目标版本为 1.2.0800。部署过程使用 fnOS 文件管理、Docker 本地镜像页面和 Docker Compose 页面完成，不要求 SSH、终端命令、容器控制台命令或宿主机安装脚本。

LeafDoc 仍然使用外部 MySQL。fnOS Compose 只运行一次性准备服务、应用服务和 Caddy 网关，不包含数据库服务。部署前必须准备一个能从应用容器访问的 MySQL 数据库，并为 LeafDoc 提供建表、读取、写入和事务所需权限。数据库地址不能填写应用容器内的 `127.0.0.1`，因为该地址指向应用容器自身。

## 文件来源和部署文件

项目源码中的 fnOS 模板是 `other/compose.fnos.yaml`。它是唯一维护来源，保存占位符和公共配置。唯一打包脚本读取该模板，并把项目名称、版本、应用镜像标签、Caddy 镜像标签以及端口表中的实际端口写入临时归档。最终归档中的文件是 `fnos/docker-compose.yml`，这个文件才是 fnOS “Compose → 新增项目”页面上传的文件。

模板中的以下文本由打包脚本自动替换：

| 模板占位符 | 生成来源 | 生成文件中的结果 | 是否由用户填写 |
| --- | --- | --- | --- |
| `__APP_IMAGE__` | 项目名称、版本、打包时间和源码摘要 | 版本化 LeafDoc 应用镜像标签 | 否 |
| `__GATEWAY_IMAGE__` | 固定的 Caddy 基础镜像版本 | Caddy 镜像标签 | 否 |
| `__FRONTEND_PORT__` | `backend/internal/core/serviceports/ports.json` | HTTPS 容器端口 | 否 |
| `__BACKEND_PORT__` | `backend/internal/core/serviceports/ports.json` | 应用内部端口 | 否 |
| `__CA_PORT__` | `backend/internal/core/serviceports/ports.json` | 公开 CA 下载容器端口 | 否 |

上传前必须使用归档中的 `fnos/docker-compose.yml`。不能把仍含有双下划线占位符的源码模板直接上传，也不能把解压后修改的生成文件反向当作下次打包的源码。需要永久修改配置时，应修改源码模板中的可填写项，重新执行唯一打包入口，再上传新归档生成的文件。

## fnOS 图形界面部署流程

在电脑上解压完整发行归档，在 fnOS 文件管理中建立专用的 LeafDoc 部署资料目录。上传归档中的 `images.tar` 和 `fnos/docker-compose.yml`。可以同时上传 `fnos/README.md`，便于在 NAS 上查看说明。

打开 fnOS 的“Docker → 本地镜像”，选择“添加镜像 → 从 NAS 文件添加”，选中 `images.tar` 并等待导入完成。导入结果中应同时出现 LeafDoc 应用镜像和 Caddy 镜像。完整的 `.tar.gz` 发行归档不是 Docker 镜像归档，不能直接在本页面导入。

在电脑的文本编辑器中修改 `fnos/docker-compose.yml` 顶部的可填写参数。将修改后的文件上传到 fnOS。打开“Docker → Compose → 新增项目”，填写项目名称，选择部署目录，选择“上传 docker-compose.yml”，上传修改后的文件，并勾选创建后立即启动。

项目启动后，在项目详情中检查三个服务。`prepare` 是一次性准备服务，成功后会以退出码零结束。`app` 和 `gateway` 应保持运行。fnOS 将一个已成功退出的 `prepare` 显示为“部分运行”属于正常情况，不能因此重复启动或删除卷。应在 `prepare` 日志中读取首次安装令牌，并只把令牌交给负责初始化的管理员。

默认局域网入口为 `https://NAS地址:24691`，公开 CA 下载入口为 `http://NAS地址:24690/api/setup/ca`。先在可信局域网下载公开 CA，核对 `prepare` 日志中的 SHA256 指纹，再在访问设备的证书管理界面导入该 CA。不要通过关闭浏览器证书校验或忽略安全警告继续使用 HTTPS。

在 HTTPS 引导页填写首次令牌、外部 MySQL 地址、端口、数据库名、账号、密码和管理员资料。提交时系统会实际检查数据库连接、建表权限、业务读写能力和管理员资料。失败时继续停留在引导页，成功后才进入正式系统。停止、启动、容器重建和镜像升级不会清除初始化状态。只有明确删除项目持久卷并重新创建实例，才会重新进入首次引导。

## Compose 配置完整说明

下面的说明以生成后的 `fnos/docker-compose.yml` 为准。源码模板中的 `__APP_IMAGE__` 等占位符在打包后会变成真实值。YAML 中以 `x-` 开头的字段是 Compose 扩展字段，用来保存可复用配置，不会单独创建容器。

### 顶层扩展字段和锚点

```yaml
x-https-port: &https-port '24691:24691'
x-ca-port: &ca-port '24690:24690'
x-settings: &settings
  LEAFDOC_HOSTS: ''
  LEAFDOC_BASE_PATH: ''
  LEAFDOC_DOMAIN: ''
  LEAFDOC_FRONTEND_PORT: '24691'
  LEAFDOC_BACKEND_PORT: '24692'
  LEAFDOC_CA_PORT: '24690'
  LEAFDOC_PUBLIC_DIR: /public-ca
  LEAFDOC_MANAGED_GATEWAY: 'true'
services:
volumes:
```

`x-https-port` 保存 HTTPS 端口映射，并用 `&https-port` 声明锚点。`x-ca-port` 保存公开 CA 端口映射，并用 `&ca-port` 声明锚点。服务中的 `*https-port` 和 `*ca-port` 是对锚点的引用，这样端口只保留一份权威值。

`x-settings` 保存应用和准备服务共同使用的环境变量，并用 `&settings` 声明锚点。`environment: *settings` 会把整个配置映射给服务。`<<: *settings` 是 YAML 合并键，表示先复制公共配置，再加入该服务自己的环境变量。公共配置只保留一份，防止准备服务和应用服务使用不同地址或端口。

`services` 是 Compose 服务定义的顶层入口。`prepare`、`app` 和 `gateway` 都必须放在它下面。`volumes` 是命名卷定义的顶层入口，卷名必须与服务挂载中的 `source` 或短语法左侧名称一致。

### 端口锚点

源码模板中的端口表达式是：

```yaml
x-https-port: &https-port '__FRONTEND_PORT__:__FRONTEND_PORT__'
x-ca-port: &ca-port '__CA_PORT__:__CA_PORT__'
```

生成后每一项都是 `宿主端口:容器端口`。左侧宿主端口可以在 fnOS 图形编辑器中修改，用来避开 fnOS 管理页面或其他应用的端口冲突。右侧容器端口由项目端口表确定，应用和网关会使用它，普通部署不应修改。

HTTPS 端口用于访问 LeafDoc。CA 端口只用于初始化阶段下载公开 CA。初始化成功后，应用会关闭 CA 下载接口，但端口映射仍可能显示在 Compose 项目中。修改左侧端口后，浏览器访问地址和 CA 下载地址必须使用新的左侧端口。修改右侧端口会同时影响环境变量、Caddy 配置和应用监听，不属于普通填写项目。

### `x-settings` 的每个键

#### `LEAFDOC_HOSTS`

默认值为空字符串。局域网或本机模式下填写实际访问地址，多个地址使用空格分隔，例如 `192.168.1.20 leafdoc.local`。这里只填写主机名或 IP，不填写 `https://`，不填写端口，不填写路径，不填写查询参数。

该值用于生成本地 HTTPS 服务器证书的地址列表。用户会通过其中一个地址访问系统，因此实际使用的地址必须出现在列表中。地址变更后，必须停止 Compose 项目，修改 YAML，再使用“更多 → 构建”重新生成配置。地址列表不需要在每次启动时修改。

#### `LEAFDOC_BASE_PATH`

默认值为空字符串，表示从根路径访问。使用反向代理路径前缀时填写类似 `/docs` 的 URL 路径。只填写路径，不填写协议、域名或端口。

填写路径前缀后，LeafDoc 入口、健康检查、公开 CA 下载地址和反向代理配置都必须使用同一前缀。例如前缀为 `/docs` 时，入口为 `https://地址:端口/docs`。不使用路径前缀时保持空值，不要填写 `/`。

#### `LEAFDOC_DOMAIN`

默认值为空字符串。局域网自签名证书模式下保持为空。公网域名模式下填写真实域名，例如 `docs.example.com`。

公网域名必须已经解析到部署入口，并且外部网络能够访问 Caddy 自动证书所需的挑战端口。公网模式下清空 `LEAFDOC_HOSTS`，由 Caddy 使用域名申请公开证书。没有真实公网域名时不要填写示例域名，也不要把局域网 IP 当作公网域名。

#### `LEAFDOC_FRONTEND_PORT`

由打包脚本从集中端口表写入，表示网关在容器内监听的 HTTPS 端口。该值必须与 `x-https-port` 的右侧端口相同。普通用户只修改端口映射左侧，不修改该环境变量。

#### `LEAFDOC_BACKEND_PORT`

由打包脚本写入，表示应用服务在容器内监听的内部端口。该端口不直接发布到 fnOS 宿主机，只供网关和应用内部使用。应用和网关必须使用同一个内部端口，不能把它改成宿主端口。

#### `LEAFDOC_CA_PORT`

由打包脚本写入，表示公开 CA 下载服务的容器端口。它必须与 `x-ca-port` 的右侧端口一致。该服务只在初始化未完成时开放有限的公开证书下载接口，不能用于下载私钥、数据库密码或管理员密码。

#### `LEAFDOC_PUBLIC_DIR`

固定为 `/public-ca`。它是容器内部公开 CA 文件目录，不是 fnOS 宿主路径。准备服务向该目录写入公开 CA，应用以只读方式挂载其中的公开证书。用户不填写，也不应改成其他目录。

#### `LEAFDOC_MANAGED_GATEWAY`

固定为字符串 `'true'`。它通知准备脚本，当前 Caddy 配置由 fnOS 的 `prepare` 服务管理。准备脚本会根据访问地址和端口生成网关配置，并在配置参数变化时保留原 CA 身份和业务数据后重新生成服务器证书。删除或修改该值会破坏网关准备逻辑。

### `prepare` 服务字段

```yaml
prepare:
  image: leafdoc:版本时间标签
  pull_policy: never
  user: '0'
  entrypoint: [sh, /prepare-instance.sh]
  environment: *settings
  volumes:
    - instance_data:/instance-data
    - instance_config:/config
    - public_certificates:/public-ca
  restart: 'no'
```

`prepare` 是实例准备服务，不是持续运行的业务服务。

`image` 必须使用打包归档中的 LeafDoc 应用镜像。该值由唯一打包脚本写入，必须与 `app.image` 完全相同。用户不能填写本地未知标签，也不能改成镜像仓库标签。

`pull_policy: never` 禁止 Compose 尝试从仓库拉取镜像。fnOS 离线部署必须先通过“本地镜像 → 添加镜像 → 从 NAS 文件添加”导入 `images.tar`，否则服务会因为找不到本地镜像而失败。

`user: '0'` 只对准备阶段有效。准备脚本需要在命名卷中创建目录、设置权限、生成单次令牌、创建本地 CA 和写入 Caddy 配置，因此使用容器内的零号用户。它不表示应用和网关以零号用户运行，业务服务仍使用镜像默认的低权限身份。

`entrypoint: [sh, /prepare-instance.sh]` 用 `sh` 执行镜像内的共享准备脚本。该脚本处理权限、令牌、访问参数摘要、本地证书、服务器证书和网关配置。不要把它替换为交互式 shell，也不要把脚本路径改成 fnOS 宿主路径。

`environment: *settings` 把公共配置传给准备脚本，使证书和 Caddy 配置使用与应用相同的地址、路径和端口。

`instance_data:/instance-data` 保存应用数据、首次安装令牌和初始化记录。`instance_config:/config` 保存 Caddyfile、访问参数摘要、证书私钥和其他包含私密内容的网关配置。`public_certificates:/public-ca` 保存可以公开下载的 CA 证书。三个卷都是命名卷，重启、重建和升级时必须保留。

`restart: 'no'` 表示准备完成后不自动重启。退出码为零表示准备成功，非零表示应用和网关不应启动。fnOS 显示项目部分运行时，应查看 `prepare` 的退出码和日志，不能只看项目总状态。

### `app` 服务字段

```yaml
app:
  image: leafdoc:版本时间标签
  pull_policy: never
  restart: unless-stopped
  depends_on:
    prepare:
      condition: service_completed_successfully
  environment:
    <<: *settings
    LEAFDOC_HOST: 0.0.0.0
    LEAFDOC_WEB_SETUP: 'true'
    LEAFDOC_DATA_DIR: /app/data
    LEAFDOC_SETUP_CA_FILE: /run/leafdoc-public/local-ca.crt
  command: [--enable-revision, --snapshot]
  volumes:
    - instance_data:/app/data
    - public_certificates:/run/leafdoc-public:ro
```

`image` 和 `prepare.image` 必须完全一致。`pull_policy: never` 的含义与准备服务相同，保证应用使用已导入的离线镜像。`restart: unless-stopped` 表示应用异常退出时由 Docker 自动恢复，管理员在 fnOS 中明确停止项目后不会被自动拉起。

`depends_on` 声明启动依赖。`prepare` 下的 `condition: service_completed_successfully` 要求准备服务以退出码零结束后，应用才可以启动。它不是简单的启动顺序，而是成功条件。数据库连接失败、证书生成失败、权限失败或配置无效时，应用会继续被阻止。

`<<: *settings` 合并全部公共环境变量。`LEAFDOC_HOST: 0.0.0.0` 让应用监听容器的所有必要接口，使网关能够访问它。`LEAFDOC_WEB_SETUP: 'true'` 保留首次初始化页面和初始化接口，同时由应用内部状态决定是否允许进入正式业务。`LEAFDOC_DATA_DIR: /app/data` 指向 `instance_data` 卷挂载点。`LEAFDOC_SETUP_CA_FILE: /run/leafdoc-public/local-ca.crt` 指向只读公开 CA 文件，应用不会因此获得含私钥的 Caddy 配置卷。

`command: [--enable-revision, --snapshot]` 向应用传入两个现有功能开关。`--enable-revision` 保留修订记录能力，`--snapshot` 保留快照和备份相关能力。该数组不是 shell 字符串，不应把两个参数合并成一个字符串，也不能删除后把功能是否可用交给 Compose 默认值。

`instance_data:/app/data` 将持久数据挂载到应用约定的数据目录。`public_certificates:/run/leafdoc-public:ro` 以只读方式提供公开 CA。`ro` 表示应用不能写入公开证书卷，也不能通过该挂载读取网关配置卷中的 CA 私钥。不要把 `instance_config` 挂载给应用，也不要删除只读标记。

### `gateway` 服务字段

```yaml
gateway:
  image: caddy:2-alpine
  pull_policy: never
  restart: unless-stopped
  depends_on:
    prepare:
      condition: service_completed_successfully
  entrypoint: [sh, -ec]
  command:
    - 'caddy fmt --overwrite /etc/caddy/Caddyfile; exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile'
  ports:
    - *https-port
    - *ca-port
  volumes:
    - type: volume
      source: instance_config
      target: /etc/caddy
      volume:
        nocopy: true
    - gateway_data:/data
    - gateway_config:/config
```

`image` 必须是归档中随 `images.tar` 导入的 Caddy 镜像。`pull_policy: never` 确保离线环境不会访问镜像仓库。`restart: unless-stopped` 让网关异常退出后自动恢复，手动停止项目后保持停止。

`depends_on` 使用与应用相同的 `service_completed_successfully` 条件。网关不会在准备失败时启动，避免使用不存在或过期的 Caddyfile。

`entrypoint: [sh, -ec]` 让容器使用 shell 执行一条严格命令。`-e` 在格式化失败时停止，`-c` 表示后续内容是要执行的命令字符串。`command` 先执行 `caddy fmt --overwrite /etc/caddy/Caddyfile`，把准备脚本生成的配置格式化，再使用 `exec caddy run` 启动 Caddy。`--config /etc/caddy/Caddyfile` 指定配置路径，`--adapter caddyfile` 指定配置格式。`exec` 让 Caddy 成为容器主进程，使停止信号和退出状态能够被 Docker 正确处理。

`ports` 发布两个端口。`*https-port` 发布 HTTPS 入口，`*ca-port` 发布初始化阶段的公开 CA 下载入口。两个引用都来自顶层锚点，不能在服务中再写一份不一致的端口。

网关的第一个挂载使用长语法。`type: volume` 表示使用 Compose 命名卷，`source: instance_config` 指定准备服务写入的配置卷，`target: /etc/caddy` 指定 Caddy 容器内的配置目录。`volume.nocopy: true` 禁止 Docker 在空卷首次挂载时把镜像内默认 Caddyfile复制进去。这个字段必须保留，否则默认站点可能被误认为已经准备完成，造成容器运行但 LeafDoc 端口无法访问。

`gateway_data:/data` 保存 Caddy 的运行数据，例如自动证书相关状态。`gateway_config:/config` 保存 Caddy 的内部配置数据。两个卷都必须在升级和重建时保留。公网模式下证书状态尤其依赖这些卷，删除后可能触发重新申请证书。

### 顶层命名卷

```yaml
volumes:
  instance_data:
  instance_config:
  public_certificates:
  gateway_data:
  gateway_config:
```

顶层卷声明使 Compose 为项目创建并管理持久卷。`instance_data` 保存页面、附件、快照、首次安装令牌和初始化记录。`instance_config` 保存网关配置、访问参数摘要、证书私钥和其他私密配置，不保存首次安装令牌。`public_certificates` 保存可公开读取的 CA 文件。`gateway_data` 保存 Caddy 运行数据。`gateway_config` 保存 Caddy 内部配置。

这些卷名是安装实例的一部分。停止和启动不会删除它们，容器重建不会删除它们，导入新镜像并重新构建项目也不应删除它们。普通卸载只停止并删除容器，必须保留卷。只有明确选择删除项目卷并重新创建项目，才会清除初始化记录并重新出现引导页面。删除前必须完成应用数据、网关配置、证书和外部 MySQL 的备份。

## 用户填写项和内部固定项

普通局域网部署只需要填写 `LEAFDOC_HOSTS`。不使用路径前缀时保持 `LEAFDOC_BASE_PATH` 为空，不使用公网域名时保持 `LEAFDOC_DOMAIN` 为空。镜像标签、应用内部端口、CA 目录、网关管理开关、服务名、依赖条件、卷名、准备脚本和容器路径均由打包方案固定，不属于用户填写项。

使用反向代理路径时，只增加 `LEAFDOC_BASE_PATH`。使用公网域名时填写 `LEAFDOC_DOMAIN`，清空 `LEAFDOC_HOSTS`，并根据实际端口条件把网关宿主映射调整为标准 80 和 443。公网 80 和 443 被 fnOS 管理界面占用时，应使用现有前置代理转发，不能强行占用管理端口。

端口冲突时优先修改映射左侧的宿主端口。右侧容器端口和环境变量保持归档中的值。地址、路径或域名发生变化时，先停止项目，再编辑 YAML，保存后使用 fnOS 的“更多 → 构建”。不要在运行中的项目中直接删除卷，也不要只更新 `app` 镜像而遗漏 `prepare` 和 `gateway` 的同一版本配置。

## 配置项填写分类总表

本节单独给出 `fnos/docker-compose.yml` 的填写分类。这里的“需要填写”是指当前部署模式下管理员必须提供有效内容，“不需要填写”是指用户不应改动，配置由打包脚本、Compose 或镜像内部提供，“可以填写也可以不填写”是指默认值已经能够完成对应功能，只有使用相应场景时才需要填写或修改。

### 需要填写的配置项

严格来说，fnOS 文件中没有要求用户填写数据库密码、管理员密码或固定令牌。那些信息在 HTTPS 首次引导页面填写，不写入 Compose 文件。Compose 文件中只有访问模式相关的配置需要在特定模式下填写。

| 配置项 | 需要填写的条件 | 应填写的内容 | 可核查结果 |
| --- | --- | --- | --- |
| `x-settings.LEAFDOC_HOSTS` | 使用本机或局域网 HTTPS 时必须填写 | NAS 的实际 IP 或主机名，多个值用空格分隔，不带协议、端口和路径 | `prepare` 日志显示准备完成，服务器证书的 SAN 包含填写地址 |
| `x-settings.LEAFDOC_DOMAIN` | 使用公网域名和 Caddy 自动证书时必须填写 | 已解析到部署入口的真实域名，不带协议、端口和路径 | 网关使用域名配置，公网证书流程能够按要求运行 |
| `x-https-port` 的宿主端口左侧 | 默认端口被 fnOS 或其他服务占用时必须修改 | 可用的宿主端口，例如将 `24691:24691` 的左侧改为其他未占用端口 | 访问 URL 使用新宿主端口，右侧容器端口仍保持归档值 |
| `x-ca-port` 的宿主端口左侧 | 默认 CA 端口被占用时必须修改 | 可用的宿主端口，例如将 `24690:24690` 的左侧改为其他未占用端口 | CA 下载地址使用新宿主端口，右侧容器端口仍保持归档值 |

`LEAFDOC_HOSTS` 和 `LEAFDOC_DOMAIN` 不是同时填写。局域网模式填写前者并保持后者为空。公网模式填写后者并清空前者。两个值都为空时，准备服务会拒绝生成可用的本地网关配置。两个值同时填写时，部署模式不明确，也必须修正后再构建。

### 不需要填写的配置项

下表中的字段在生成的 `fnos/docker-compose.yml` 中必须存在，但管理员不应填写、替换或删除。它们要么由打包脚本生成，要么是 Compose 结构，要么是应用和网关之间必须保持一致的内部约束。

| 配置项 | 不需要填写的原因 | 核查要求 |
| --- | --- | --- |
| `x-settings` | 公共环境变量锚点定义 | 必须保留，服务通过别名复用 |
| `services` | Compose 服务集合入口 | 必须包含 `prepare`、`app`、`gateway` |
| 顶层 `volumes` | Compose 命名卷集合入口 | 必须包含五个项目卷 |
| `prepare`、`app`、`gateway` | 服务名称由依赖关系和管理界面使用 | 不得改名 |
| `prepare.image` | 打包脚本写入当前应用镜像标签 | 必须与 `app.image` 相同 |
| `app.image` | 打包脚本写入当前应用镜像标签 | 必须与 `prepare.image` 相同 |
| `gateway.image` | 打包脚本写入 Caddy 镜像标签 | 必须与 `images.tar` 导入的 Caddy 镜像相同 |
| `prepare.pull_policy`、`app.pull_policy`、`gateway.pull_policy` | 离线部署必须禁止拉取镜像 | 必须保持 `never` |
| `prepare.user` | 准备阶段需要初始化卷权限 | 必须保持 `'0'`，只限一次性准备服务 |
| `prepare.entrypoint` | 指向镜像内的共享准备脚本 | 必须保持 `[sh, /prepare-instance.sh]` |
| `prepare.environment` | 引用公共设置 | 必须保持 `*settings` |
| `prepare.volumes` | 连接三个准备阶段命名卷 | 三个挂载必须保留 |
| `prepare.restart` | 准备服务只执行一次 | 必须保持 `'no'` |
| `app.restart`、`gateway.restart` | 业务服务异常时由 Docker 恢复 | 必须保持 `unless-stopped` |
| `app.depends_on.prepare`、`gateway.depends_on.prepare` | 声明准备服务依赖 | 两个服务都必须依赖 `prepare` |
| 两个 `condition` | 阻止准备失败时提前启动业务 | 必须保持 `service_completed_successfully` |
| `app.environment`、`gateway.environment` | 服务环境映射结构 | 必须保留相应服务的全部键 |
| `app.environment.<<` | 合并公共配置 | 必须保持 `*settings` |
| `app.LEAFDOC_HOST` | 容器内监听地址 | 必须保持 `0.0.0.0` |
| `app.LEAFDOC_WEB_SETUP` | 保留初始化阶段接口，由应用状态控制门禁 | 必须保持 `'true'` |
| `app.LEAFDOC_DATA_DIR` | 应用数据目录 | 必须保持 `/app/data`，与数据卷目标一致 |
| `app.LEAFDOC_SETUP_CA_FILE` | 应用读取公开 CA 的位置 | 必须保持 `/run/leafdoc-public/local-ca.crt` |
| `app.command`、`--enable-revision`、`--snapshot` | 保留修订和快照能力 | 参数数组必须完整保留 |
| `app.volumes` | 应用数据和公开 CA 挂载 | 数据卷和只读 CA 卷必须保留 |
| `gateway.entrypoint` | 以严格 shell 方式执行 Caddy 命令 | 必须保持 `[sh, -ec]` |
| `gateway.command` | 格式化并启动 Caddy | 必须保留 `caddy fmt`、`exec caddy run`、配置路径和适配器参数 |
| `gateway.ports` | 网关发布入口 | 必须保留两个锚点引用 |
| `gateway.volumes.type` | 长语法挂载类型 | 必须保持 `volume` |
| `gateway.volumes.source` | 指向准备服务创建的配置卷 | 必须保持 `instance_config` |
| `gateway.volumes.target` | Caddy 容器配置目录 | 必须保持 `/etc/caddy` |
| `gateway.volumes.volume` | 长语法挂载选项集合 | 必须保留 |
| `gateway.volumes.volume.nocopy` | 阻止默认 Caddyfile 污染空卷 | 必须保持 `true` |
| `gateway_data:/data`、`gateway_config:/config` | Caddy 运行数据和内部配置 | 必须保留，不能改为临时目录 |
| 顶层 `volumes.instance_data` | 应用数据卷声明 | 必须保留原名 |
| 顶层 `volumes.instance_config` | 网关配置和私密证书卷声明 | 必须保留原名 |
| 顶层 `volumes.public_certificates` | 公开 CA 卷声明 | 必须保留原名 |
| 顶层 `volumes.gateway_data` | Caddy 数据卷声明 | 必须保留原名 |
| 顶层 `volumes.gateway_config` | Caddy 配置数据卷声明 | 必须保留原名 |

源码模板中的 `__APP_IMAGE__`、`__GATEWAY_IMAGE__`、`__FRONTEND_PORT__`、`__BACKEND_PORT__` 和 `__CA_PORT__` 也属于不需要填写的内容。它们不是 fnOS 用户配置，而是唯一打包入口的生成输入。生成归档中不应再出现这些占位符。

### 可以填写也可以不填写的配置项

| 配置项 | 默认行为 | 需要填写或修改的情况 | 可以省略的情况 |
| --- | --- | --- | --- |
| `x-settings.LEAFDOC_BASE_PATH` | 空字符串，从根路径访问 | 通过反向代理以 `/docs` 等路径前缀发布时填写 | 直接使用域名、IP 和端口访问时省略 |
| `x-https-port` 左侧宿主端口 | 使用打包文件提供的 HTTPS 宿主端口 | 宿主端口冲突或需要统一端口规划时修改 | 默认端口未被占用时保留 |
| `x-ca-port` 左侧宿主端口 | 使用打包文件提供的 CA 宿主端口 | CA 端口冲突时修改 | 默认端口未被占用时保留 |
| `LEAFDOC_HOSTS` 中的附加地址 | 只填写一个实际入口也能运行 | 需要通过多个局域网主机名或 IP 访问时增加 | 只有一个访问地址时不增加 |
| `LEAFDOC_DOMAIN` | 空字符串，使用本地证书路径 | 已有真实公网域名并准备使用公开证书时填写 | 没有公网域名或只在局域网运行时省略 |

`LEAFDOC_DOMAIN` 虽然在公网模式下属于必须填写项，但在整个项目的局域网部署范围内可以省略。因此它在表格中同时具有条件必填和场景可省略两个属性。判断标准不是是否存在该键，而是当前部署模式是否需要公网自动证书。

### 全文件键路径核对清单

下面的清单覆盖生成文件中的结构键、配置键、服务字段、嵌套键和挂载项。核对文件时应按路径检查，不能只检查服务名称。

| 键路径 | 分类 | 处理方式 |
| --- | --- | --- |
| `x-https-port` | 可选填写左侧宿主端口，右侧固定 | 只在端口冲突时修改左侧 |
| `x-https-port.&https-port` | 不需要填写 | 保留端口锚点名称 |
| `x-ca-port` | 可选填写左侧宿主端口，右侧固定 | 只在端口冲突时修改左侧 |
| `x-ca-port.&ca-port` | 不需要填写 | 保留端口锚点名称 |
| `x-settings` | 不需要填写 | 保留锚点定义 |
| `x-settings.&settings` | 不需要填写 | 保留公共环境变量锚点名称 |
| `x-settings.LEAFDOC_HOSTS` | 局域网模式必填，公网模式留空 | 填主机名或 IP |
| `x-settings.LEAFDOC_BASE_PATH` | 可填可不填 | 只在路径前缀模式填写 |
| `x-settings.LEAFDOC_DOMAIN` | 公网模式必填，局域网模式留空 | 填真实域名 |
| `x-settings.LEAFDOC_FRONTEND_PORT` | 不需要填写 | 保留打包脚本生成值 |
| `x-settings.LEAFDOC_BACKEND_PORT` | 不需要填写 | 保留打包脚本生成值 |
| `x-settings.LEAFDOC_CA_PORT` | 不需要填写 | 保留打包脚本生成值 |
| `x-settings.LEAFDOC_PUBLIC_DIR` | 不需要填写 | 保留 `/public-ca` |
| `x-settings.LEAFDOC_MANAGED_GATEWAY` | 不需要填写 | 保留 `'true'` |
| `services` | 不需要填写 | 保留服务集合 |
| `services.prepare` | 不需要填写 | 保留服务名 |
| `services.prepare.image` | 不需要填写 | 使用打包生成的应用镜像 |
| `services.prepare.pull_policy` | 不需要填写 | 保留 `never` |
| `services.prepare.user` | 不需要填写 | 保留 `'0'` |
| `services.prepare.entrypoint` | 不需要填写 | 保留准备脚本入口 |
| `services.prepare.environment` | 不需要填写 | 保留 `*settings` |
| `services.prepare.environment.*settings` | 不需要填写 | 保留公共设置引用 |
| `services.prepare.volumes` | 不需要填写 | 保留三个准备卷挂载 |
| `services.prepare.volumes.instance_data:/instance-data` | 不需要填写 | 保留应用数据准备挂载 |
| `services.prepare.volumes.instance_config:/config` | 不需要填写 | 保留网关配置准备挂载 |
| `services.prepare.volumes.public_certificates:/public-ca` | 不需要填写 | 保留公开 CA 准备挂载 |
| `services.prepare.restart` | 不需要填写 | 保留 `'no'` |
| `services.app` | 不需要填写 | 保留服务名 |
| `services.app.image` | 不需要填写 | 使用与 `prepare` 相同的应用镜像 |
| `services.app.pull_policy` | 不需要填写 | 保留 `never` |
| `services.app.restart` | 不需要填写 | 保留 `unless-stopped` |
| `services.app.depends_on.prepare` | 不需要填写 | 保留准备服务依赖 |
| `services.app.depends_on.prepare.condition` | 不需要填写 | 保留 `service_completed_successfully` |
| `services.app.environment` | 不需要填写 | 保留环境映射结构 |
| `services.app.environment.<<` | 不需要填写 | 保留公共设置合并 |
| `services.app.environment.LEAFDOC_HOST` | 不需要填写 | 保留 `0.0.0.0` |
| `services.app.environment.LEAFDOC_WEB_SETUP` | 不需要填写 | 保留 `'true'` |
| `services.app.environment.LEAFDOC_DATA_DIR` | 不需要填写 | 保留 `/app/data` |
| `services.app.environment.LEAFDOC_SETUP_CA_FILE` | 不需要填写 | 保留公开 CA 文件路径 |
| `services.app.command` | 不需要填写 | 保留两个应用开关 |
| `services.app.command.--enable-revision` | 不需要填写 | 保留修订功能开关 |
| `services.app.command.--snapshot` | 不需要填写 | 保留快照功能开关 |
| `services.app.volumes` | 不需要填写 | 保留数据卷和只读 CA 卷 |
| `services.app.volumes.instance_data:/app/data` | 不需要填写 | 保留应用数据挂载 |
| `services.app.volumes.public_certificates:/run/leafdoc-public:ro` | 不需要填写 | 保留公开 CA 只读挂载 |
| `services.gateway` | 不需要填写 | 保留服务名 |
| `services.gateway.image` | 不需要填写 | 使用归档导入的 Caddy 镜像 |
| `services.gateway.pull_policy` | 不需要填写 | 保留 `never` |
| `services.gateway.restart` | 不需要填写 | 保留 `unless-stopped` |
| `services.gateway.depends_on.prepare` | 不需要填写 | 保留准备服务依赖 |
| `services.gateway.depends_on.prepare.condition` | 不需要填写 | 保留 `service_completed_successfully` |
| `services.gateway.entrypoint` | 不需要填写 | 保留 `[sh, -ec]` |
| `services.gateway.command` | 不需要填写 | 保留完整 Caddy 命令 |
| `services.gateway.command.caddy fmt` | 不需要填写 | 保留配置格式化步骤 |
| `services.gateway.command.exec caddy run` | 不需要填写 | 保留 Caddy 主进程启动步骤 |
| `services.gateway.ports` | 不需要填写 | 保留两个端口锚点引用 |
| `services.gateway.ports.*https-port` | 不需要填写 | 保留 HTTPS 端口引用 |
| `services.gateway.ports.*ca-port` | 不需要填写 | 保留公开 CA 端口引用 |
| `services.gateway.volumes.type` | 不需要填写 | 保留 `volume` |
| `services.gateway.volumes.source` | 不需要填写 | 保留 `instance_config` |
| `services.gateway.volumes.target` | 不需要填写 | 保留 `/etc/caddy` |
| `services.gateway.volumes.volume.nocopy` | 不需要填写 | 保留 `true` |
| `services.gateway.volumes.instance_config:/etc/caddy` | 不需要填写 | 保留 Caddy 配置挂载 |
| `services.gateway.volumes.gateway_data:/data` | 不需要填写 | 保留 Caddy 数据挂载 |
| `services.gateway.volumes.gateway_config:/config` | 不需要填写 | 保留 Caddy 内部配置挂载 |
| `volumes.instance_data` | 不需要填写 | 保留命名卷 |
| `volumes.instance_config` | 不需要填写 | 保留命名卷 |
| `volumes.public_certificates` | 不需要填写 | 保留命名卷 |
| `volumes.gateway_data` | 不需要填写 | 保留命名卷 |
| `volumes.gateway_config` | 不需要填写 | 保留命名卷 |

这份清单中的“必须填写”是条件化的实际用户输入，“可填可不填”是场景选择，“不需要填写”不是允许删除，而是要求保持打包生成的内容不变。用户看到空字符串时，也要先根据部署模式判断该项是有意留空还是缺少必填地址。

## 其他 Compose 文件及选择方式

项目中存在多个服务场景文件，它们用途不同，不能互换。

| 文件 | 用途 | 是否上传到 fnOS |
| --- | --- | --- |
| `other/compose.fnos.yaml` | fnOS 源码模板，生成 `fnos/docker-compose.yml` | 否，不能直接上传未替换的模板 |
| 归档 `fnos/docker-compose.yml` | fnOS 离线图形部署文件 | 是 |
| `other/compose.bundle.yaml` | Ubuntu 离线安装模板，生成归档根目录 `compose.yaml` | 否 |
| `other/compose.yaml` | 源码开发和 Ubuntu 完整服务栈，包含 MySQL 服务 | 否 |
| `other/compose.public.yaml` | Ubuntu 公网模式的端口覆盖配置 | 否，不能作为 fnOS 默认文件 |
| 归档根目录 `compose.yaml` | Ubuntu 安装入口使用的生成文件 | 否，fnOS 页面应使用 `fnos/docker-compose.yml` |

`compose.yaml` 是 Compose 默认认可的现代文件名，`docker-compose.yml` 是兼容旧版 Compose 和当前 fnOS 上传界面的文件名。两者都可以描述 `services`、`volumes`、`ports`、`environment`、`depends_on` 和挂载，但文件名不会改变服务实际行为。当前 fnOS 图形界面要求上传 `docker-compose.yml`，因此归档内固定使用这个名称。

`other/compose.yaml` 的 `database` 服务属于源码或 Ubuntu 完整服务栈，使用 MySQL 镜像和数据库卷。fnOS 方案明确连接外部 MySQL，因此不能把该文件上传到 fnOS，也不能把 `database` 服务复制到 fnOS 文件中。`other/compose.bundle.yaml` 是 Ubuntu 离线安装资料的模板，使用宿主机目录 `/etc/leafdoc` 和 `/var/lib/leafdoc`，与 fnOS 的命名卷结构不同。

`other/compose.public.yaml` 是 Ubuntu 公网模式覆盖文件，增加宿主机 80、443 的 TCP 映射和 443 的 UDP 映射。它不改变 fnOS 默认局域网模式，也不能替代 fnOS 文件顶部的地址配置。源码目录和测试目录中可能存在临时 Compose 文件，它们只用于验证，不是正式部署入口。

### `compose.yaml` 的结构和作用

项目中出现的 `compose.yaml` 有两个层次，必须按所在位置理解。

归档根目录中的 `compose.yaml` 是由 `other/compose.bundle.yaml` 生成的 Ubuntu 离线服务文件。它只包含 `app` 和 `gateway`，不包含数据库服务。Ubuntu 安装入口会生成 `/etc/leafdoc/environment`，再用 `--env-file` 把镜像标签、端口、数据目录和可选运行参数提供给 Compose。

归档根目录 `compose.yaml` 的主要字段如下：

| 键路径 | 作用 | 是否用于 fnOS |
| --- | --- | --- |
| `services.app.image` | 使用打包生成的 LeafDoc 应用镜像，使用必填变量表达式防止标签缺失 | 否 |
| `services.app.pull_policy` | 设为 `never`，要求使用本地导入镜像 | 否 |
| `services.app.restart` | 使用 `unless-stopped` 保持 Ubuntu 应用服务 | 否 |
| `services.app.env_file` | 读取 `/etc/leafdoc/application.env` 中的可选运行配置 | 否 |
| `services.app.environment.LEAFDOC_HOST` | 让应用监听容器接口 | 否 |
| `services.app.environment.LEAFDOC_WEB_SETUP` | 保留网页初始化入口 | 否 |
| `services.app.environment.LEAFDOC_DATA_DIR` | 指向 `/app/data` | 否 |
| `services.app.environment.LEAFDOC_BASE_PATH` | 传入可选路径前缀 | 否 |
| `services.app.command` | 启用修订和快照功能 | 否 |
| `services.app.volumes` | 把宿主机 `/var/lib/leafdoc` 挂载为应用数据目录 | 否 |
| `services.database.healthcheck` | Ubuntu 内置 MySQL 的健康检查，应用只在数据库健康后启动 | 否 |
| `services.database.healthcheck.test` | 使用容器内账号执行 `SELECT 1` | 否 |
| `services.database.healthcheck.interval`、`timeout`、`retries`、`start_period` | 控制 MySQL 健康检查频率、单次超时、失败次数和启动宽限期 | 否 |
| `services.app.stop_grace_period` | 为应用提供优雅停止时间 | 否 |
| `services.gateway.image` | 使用打包生成的 Caddy 镜像 | 否 |
| `services.gateway.pull_policy` | 禁止离线部署时拉取镜像 | 否 |
| `services.gateway.restart` | 保持 Ubuntu 网关服务运行 | 否 |
| `services.gateway.ports` | 发布应用端口以及标准 HTTP 和 HTTPS 入口 | 否 |
| `services.gateway.volumes` | 挂载 Ubuntu 的 Caddyfile、TLS 目录和 Caddy 命名卷 | 否 |
| `services.gateway.stop_grace_period` | 为网关提供优雅停止时间 | 否 |
| `services.database.logging`、`services.app.logging`、`services.gateway.logging` | 使用 Docker `json-file` 日志驱动并限制日志大小和文件数 | 否 |
| `logging.driver`、`logging.options.max-size`、`logging.options.max-file` | Ubuntu 服务的日志轮转设置 | 否 |
| 顶层 `volumes.gateway_data` | 保存 Caddy 运行数据 | 否 |
| 顶层 `volumes.gateway_config` | 保存 Caddy 内部配置 | 否 |

该文件的数据库连接来自应用首次初始化和 Ubuntu 安装环境，不等同于 fnOS 文件。fnOS 使用命名卷 `instance_data`、`instance_config` 和 `public_certificates`，并由 `prepare` 服务创建证书和网关配置。Ubuntu 文件使用宿主机 `/etc/leafdoc`、`/var/lib/leafdoc`，两套挂载方式不能交叉复制。

源码中的 `other/compose.yaml` 是源码开发和 Ubuntu 完整服务栈文件。它额外定义 `database` 服务，包含 MySQL 镜像、数据库环境变量、数据库卷、健康检查和日志轮转。`app` 通过 `depends_on.database.condition: service_healthy` 等待 MySQL 健康，应用使用 `database` 作为数据库主机。该文件适合源码环境或需要 Compose 内置 MySQL 的 Ubuntu 场景，不适合 fnOS，因为 fnOS 方案明确要求连接外部 MySQL，不能把数据库镜像带入 fnOS 发行包。

`other/compose.bundle.yaml` 是 Ubuntu 离线归档的模板。它与源码完整服务栈的区别是删除了 `database` 服务，改用外部数据库和安装目录，并通过 `env_file` 读取安装后的私有配置。维护 Ubuntu 离线部署时修改该模板，维护 fnOS 部署时修改 `other/compose.fnos.yaml`。

`other/compose.public.yaml` 是 Ubuntu 公网模式覆盖文件。它只覆盖 `gateway.ports`，增加宿主机 80 和 443 的 TCP 映射以及 443 的 UDP 映射，让 Caddy 支持标准公网入口和 HTTP/3。它不是独立服务栈，必须与 Ubuntu 的基础 Compose 文件一起使用。fnOS 公网模式的端口修改直接在 fnOS 专用 YAML 中完成，不上传该覆盖文件。

项目目前没有把 `compose.override.yaml`、Compose profile 或数据库初始化 Compose 文件作为正式部署入口。不要自行创建第二份同用途配置来绕开现有文件。需要增加部署场景时，应保持基础服务、卷名、初始化门禁和镜像来源的单一约束，并同步更新本说明。

### 文件名、环境文件和打包归档的关系

`compose.yaml` 是现代 Compose 默认文件名，`docker-compose.yml` 是旧版 Compose 和当前 fnOS 图形界面常用文件名。文件名只影响 Compose 或图形界面默认查找的文件，不会改变 YAML 语义。fnOS 页面当前要求选择 `docker-compose.yml`，因此归档内使用该名称。Ubuntu 安装入口明确传入归档根目录 `compose.yaml`，不依赖默认文件名查找。

fnOS 文件不使用独立 `.env` 或 `env_file`。打包脚本把实际镜像标签和内部端口写入文件，管理员只修改顶部访问参数和必要的宿主端口左侧。Ubuntu 文件使用安装入口生成的 `release.env`、`environment` 和 `application.env`，其中数据库密码、JWT 密钥和其他私密运行参数不进入 Compose 模板。不要把 Ubuntu 的环境文件上传到 fnOS，也不要把数据库密码写进 fnOS YAML。

安装归档同时包含 `images.tar`、Compose 文件、安装入口、`release.env`、`manifest.json`、`SHA256SUMS` 和部署说明。`images.tar` 负责离线镜像导入，Compose 文件负责服务拓扑和挂载，`manifest.json` 记录版本、平台、镜像和源码摘要，`SHA256SUMS` 校验归档内部文件。它们共同组成部署资料，单独上传一个 Compose 文件不能替代镜像导入和配置准备。

## 升级、停止和故障处理

升级必须从新的发行归档中重新导入 `images.tar`，在原项目中停止服务，上传并保存新的 `fnos/docker-compose.yml`，然后使用 Compose 页面的“构建”或同等可视化更新操作。项目名称、部署目录、访问参数和五个命名卷必须保持不变。`prepare`、`app` 和 `gateway` 必须同时使用同一次打包生成的版本标签。

普通停止和启动在 Compose 项目页面完成。日志在项目详情或容器详情查看。准备失败时先读取 `prepare` 日志中的具体错误，修正地址、端口或目录配置后重新构建，不要删除卷来绕过错误。数据库错误、证书错误和权限错误都必须保留原实例状态，不会自动重置首次引导。

无法访问 HTTPS 时，检查宿主端口左侧映射、访问地址是否在 `LEAFDOC_HOSTS` 中、客户端是否信任公开 CA、Caddy 日志是否显示配置错误，以及 fnOS 管理服务是否占用端口。CA 下载失败时只能从初始化未完成的实例获取，并核对日志指纹。初始化成功后下载接口关闭属于预期行为。

数据库密码、管理员密码、首次令牌、CA 私钥和 Caddy 私密配置不能写入 Compose 文件、截图、公开文档或 Git。Compose 文件只保存部署所需的非敏感连接位置和运行参数，数据库凭据由 HTTPS 初始化页面提交并持久化保存。

## 验收与维护检查

每次修改模板或文档后，应重新执行唯一打包入口，确认 `installers` 目录只保留本次归档。解压归档后检查 `fnos/docker-compose.yml` 已经替换全部占位符，检查 `fnos/README.md` 与源码文档内容一致，使用 Compose 配置解析检查 YAML 语法和服务结构，并核对 `SHA256SUMS`。

维护人员应逐项检查顶层扩展字段、端口锚点、八个公共环境变量、三个服务及其全部子键、五个命名卷和相关挂载。还应确认 `pull_policy: never`、`service_completed_successfully`、`nocopy: true`、只读公开 CA 挂载以及 `prepare` 的退出状态仍然存在。

实际部署验收必须包含镜像离线导入、Compose 创建、准备服务成功、首次令牌读取、CA 信任、空字段拒绝、错误数据库拒绝、成功初始化、管理员登录、停止、重启、容器重建、升级和原数据恢复。未实际运行的宿主系统、公网证书签发和外部集成必须单独标记为未验证，不能用 Compose 文件解析成功代替完整运行验证。
