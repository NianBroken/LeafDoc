# fnOS 可视化 Docker 部署

## 发行包与准备

使用 package.mjs 生成的完整发行包。解压并校验 SHA256SUMS，执行 docker load -i images.tar。fnos/docker-compose.yml 已由构建脚本替换镜像和固定端口占位符，直接使用生成文件，不把仓库中的 __APP_IMAGE__ 等占位符作为真实镜像启动。

本模板不带数据库，准备应用容器可达的 MySQL 与凭据。与其他 LeafDoc 部署共存时检查端口，避免相互抢占。

## 可视化创建

在 fnOS Docker 项目界面创建项目，使用包内 fnos/docker-compose.yml，保持项目名稳定。x-settings 的 LEAFDOC_HOSTS 填实际访问地址，多个地址空格分隔。BASE_PATH 是可选 URL 前缀，DOMAIN 参与证书地址准备，不等于自动获取公网受信任证书。

prepare 使用 root 完成卷权限、证书、安装令牌和设置准备，完成成功后 app 与 gateway 才启动。prepare 是一次性服务，Exited 0 是正常结果，不要求它常驻。

| 卷 | 用途 |
| --- | --- |
| instance_data | 安装状态、附件等实例文件 |
| instance_config | TLS、私钥及准备配置 |
| public_certificates | 可供安装流程读取的公开 CA |

应用只读挂载配置与公开证书，持久数据读写。不要删除卷更新程序。

## 安装与访问

启动后检查 prepare 成功以及 app、gateway 运行。访问服务器 24691 端口，HTTP 与 HTTPS 同端口。首次按引导页令牌流程连接 MySQL、配置管理员并完成安装。令牌保存在 instance_data 卷的 setup-token，通过容器管理界面查看对应挂载目录，不把令牌写入模板。

HTTPS 信任安装流程提供的 CA，并使用证书匹配地址。完成后实际登录，打开教程并保存练习页。

## 更新和长期运行

导入新包 images.tar，用新生成模板中的镜像标签更新原项目，保持项目名与卷名。确认原 x-settings 访问地址和前缀保留，再重建 app、gateway。prepare 按依赖执行设置准备，不能更换成不关联旧卷的新项目。

app 和 gateway 的 unless-stopped 保持后台恢复，prepare 的 restart 为 no。宿主 Docker 服务还需开机运行。检查日志、访问与原管理员登录后确认更新，主动停止的容器需要手动恢复。

完整镜像打包、校验和字段解释分别见包内主 README 及源码配置说明。失效链接按实际页面路径排查，数据库不可达时检查容器网络地址，不使用容器 localhost 指代宿主。
