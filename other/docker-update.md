# Docker 新代码发行与原环境更新

## 适用环境

本流程用于离线包安装的 Ubuntu，配置在 /etc/leafdoc，服务在 /opt/leafdoc，实例文件在 /var/lib/leafdoc。源码 Compose 使用 /etc/leafdoc-compose，不混用入口。

更新前记录当前 manage.sh status、manifest、镜像标签与数据库连接方式。生产维护应备份数据库和实例文件并确认恢复能力。本次明确授权的文档清空不是普通升级步骤。

## 新代码打包与上传

构建机仓库根目录执行：

```sh
git pull --ff-only
git status --short
node other/scripts/package.mjs
```

源码摘要包含未提交且未忽略的改动，核对 manifest 的 sourceHash、构建时间、镜像标签和逐文件摘要，不只看版本号。将新包用 scp 上传到原 Ubuntu 的独立版本目录，核对整体 SHA256，解压并执行 sha256sum -c SHA256SUMS，保留旧包。

## 升级顺序

确认数据库可用后，在新包目录执行：

```sh
sudo /opt/leafdoc/manage.sh status
sha256sum -c SHA256SUMS
sudo sh install.sh upgrade
sudo /opt/leafdoc/manage.sh status
sudo /opt/leafdoc/manage.sh logs
curl --fail http://127.0.0.1:24691/api/health
```

脚本校验和加载新镜像，替换系统 Compose、管理脚本、端口脚本与 manifest。已有 environment 只替换 LEAFDOC_IMAGE 行，其余参数保留。已有 application.env 不覆盖，实例目录、外部数据库、安装状态和既有证书保留。

Compose up -d 根据新镜像重建服务，会有切换时间，不是无中断滚动更新。无需先 down，不删除卷，不 purge，不重新完成首次安装。修改已有部署参数应编辑实际环境文件，核对 Compose 配置后重新创建容器。

## 验证更新

| 检查 | 操作与成功结果 |
| --- | --- |
| 来源 | /opt/leafdoc/manifest.json 与新包摘要和镜像标签一致 |
| 镜像 | docker inspect 检查 app 和 gateway 镜像 ID，与新标签对应 |
| 访问 | 健康接口成功，浏览器首页和静态资源完整加载 |
| 登录 | 原管理员能进入系统，不进入首次安装 |
| 数据 | 原页面与附件可访问 |
| 改动 | 亲自执行本次代码改变对应功能，观察实际结果 |
| 保活 | restart policy 仍为 unless-stopped |
| 自启 | systemctl is-enabled docker 返回 enabled |

有前缀时修改健康地址，登录使用同一入口。缓存问题检查资源请求并重新加载，不把清缓存当作服务器更新。

## 失败恢复

校验失败重新传输，加载失败处理空间和归档，启动失败检查参数、证书权限、数据库及具体日志。保留失败现场，不删除数据解决启动错误。

回退程序时进入保留的旧包目录，再执行 sudo sh install.sh upgrade。程序回退不等于数据库回退，存在数据结构变化时使用经过验证的配套恢复流程，不承诺旧程序直接兼容新数据。

确认业务稳定后按明确镜像 ID 清理不再使用的旧镜像，不执行无差别 volume prune，不删除唯一旧包与恢复资料。
