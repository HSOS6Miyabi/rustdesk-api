# RustDesk API

> **版本与维护资料：** [版本说明](docs/HANDOFF_SNAPSHOT_2026-11-09.md)、[实施台账](docs/IMPLEMENTATION_LEDGER.md)、[T001–T022 检查记录](docs/TASK_VERIFICATION_RECORD.md)。正式版本：[RustDesk API v3.0.0-api.105](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.105)、[RustDesk Server v1.1.17-api.21](https://github.com/HSOS6Miyabi/rustdesk-server/releases/tag/v1.1.17-api.21)。API 与 Server 的主分支代码已经合并，两个版本的 Linux AMD64、ARM64 Docker 镜像已经发布。Server 组合镜像使用 API v3.0.0-api.103。

基于 [lejianwen/rustdesk-api](https://github.com/lejianwen/rustdesk-api) 继续维护的 RustDesk 自托管 API 与管理控制台。可与本项目的 [RustDesk Server](https://github.com/HSOS6Miyabi/rustdesk-server) 配合使用。

本项目提供设备和用户管理、地址簿、联系人共享、远程访问记录、OIDC/SSO 登录、WebClient v2，以及适合电脑和平板、手机使用的管理界面。

## 主要功能

- **控制台**：在 `/_admin/` 登录。可管理用户、设备、群组、地址簿、联系人、共享设备和访问记录。移动端可使用完整管理功能。
- **地址簿**：同一个页面内管理地址簿分类及其中的设备。原有数据保留。
- **普通组**：成员只能在 RustDesk 客户端中查看自己的账户及设备。
- **共享组**：成员可以在 RustDesk 客户端中查看同组其他成员及其设备。
- **公共锚点组**：成员只能看到自己和管理员指定的公共成员及其设备。即使组内的某位成员是系统管理员，其 RustDesk 客户端也遵守相同的可见范围。管理员在后台仍可维护所有设备。公共成员可以随时修改。
- **个人群组**：所有用户都可通过“我的群组”创建普通组、共享组或公共锚点组，并管理自己创建的群组及其成员。管理员通过独立的“用户分组”页面管理全部群组，管理操作在服务端检查群组归属。
- **个人审计记录**：普通用户可以分别查看自己发起的远程连接、其他设备连接自己设备的记录，以及与本人设备有关的文件传输记录。日志记录连接方和被连接方的设备 ID、设备名称、账户和连接时间；原有记录会在读取时尽可能补充信息。
- **群组成员**：一个用户可以保留主要分组并加入其他分组。管理员可批量添加、移除或替换成员。
- **OIDC/SSO**：可以为每个身份认证服务指定自动加入的一个或多个分组。用户通过该服务登录后自动加入。
- **联系人**：A 添加 B 为联系人后，B 即可在共享设备页面查看 A 的全部设备，无须确认，也不会自动向 A 共享 B 的设备。A 可以随时撤销共享。
- **共享设备**：集中显示通过联系人、共享组和公共锚点组公开给自己的设备。支持按共享来源筛选、搜索并显示全部共享设备；同一设备只显示一次，并标明全部来源。
- **WebClient v2**：在 `/webclient2/` 使用浏览器连接 RustDesk 设备，支持共享设备选择、WebRTC 直接连接选项和 WebSocket 中继后备连接。
- **安全审计**：保存连接双方账户和设备、连接认证方式、双重认证方式、文件传输连接编号和安全告警。
- **日志管理**：管理员可查看管理操作记录、导出 CSV，并为五类日志分别设置保留期限。
- **设备安全策略**：管理员可设置设备 ID 白名单、IP 白名单和可信设备有效期。策略分配后，客户端通过心跳请求自动取得相应设置。WebRTC 开关由客户端本地设置控制。

群组和联系人只控制 API 返回哪些账户与设备。即使某个设备未出现在列表中，用户在获得设备 ID 和必要的连接凭据后，仍可由 RustDesk 客户端直接发起连接。

## 运行环境

建议使用 Go 1.23 或更新版本。数据库支持 SQLite、MySQL 和 PostgreSQL。默认使用 SQLite，不需要单独安装数据库服务。

在服务器上执行：

```bash
git clone https://github.com/HSOS6Miyabi/rustdesk-api.git
cd rustdesk-api
cd resources/web/js
npm ci
npm run build
cd ../../..
go build -o apimain ./cmd/apimain.go
mkdir -p data runtime
./apimain
```

默认 HTTP 端口为 `21114`。首次启动时，会在服务端日志显示管理员 `admin` 的初始密码。登录后请立即修改密码，也可以运行 `./apimain reset-admin-pwd 新密码`。

打开 `http://服务器地址:21114/_admin/` 进入控制台。公网部署请通过 HTTPS 访问。

Docker 构建前也需要先运行上面的 `npm ci` 与 `npm run build`，以生成 WebClient v2 连接组件。直接使用 GitHub Release 或 GHCR 镜像时，所需网页文件已包含在发布内容中。

Docker 构建与运行：

```bash
docker build -t rustdesk-api .
docker run -d --name rustdesk-api -p 21114:21114 \
  -v "$PWD/data:/app/data" \
  --restart unless-stopped rustdesk-api
```

Docker 镜像也通过 GitHub Container Registry 发布：`ghcr.io/hsos6miyabi/rustdesk-api`。

详细配置见 [部署与配置](docs/DEPLOYMENT.md)，接口说明见 [API 使用说明](docs/API.md)。

## 与 RustDesk Server 配合使用

在 `conf/config.yaml` 中填写 ID 服务器、中继服务器、公钥和 API 访问地址。建议同时为 API 和 RustDesk Server 设置同一个 `RUSTDESK_API_JWT_KEY`，用于验证用户登录令牌。

服务器原有的 `MUST_LOGIN` 设置不受联系人和群组可见范围影响。需要所有连接都先登录时，再将其设置为 `Y`。

## 更新现有安装

升级之前备份 `data/rustdeskapi.db`，MySQL 或 PostgreSQL 用户请备份对应数据库。新版本会自动添加群组成员关系、公共成员以及联系人等数据库表。现有用户、设备、地址簿和访问记录会保留。

## 构建、测试及发布

GitHub Actions 包含 Go、JavaScript、浏览器交互、可执行文件编译及 Docker 镜像构建测试。日常提交与拉取请求不会自动发布版本。需要发布 `v3.0.0-api.105` 时，在 GitHub Actions 打开 [Publish approved RustDesk API milestone](https://github.com/HSOS6Miyabi/rustdesk-api/actions/workflows/publish-approved-milestone.yml)，选择 `master` 分支，勾选 `milestone_release` 后运行。工作流会创建版本标签，并启动 `Verify and release RustDesk API`；测试与编译成功后，会发布 Linux AMD64、Linux ARM64、Windows AMD64 安装包和 GHCR 多架构 Docker 镜像。

查看 [维护说明](docs/MAINTENANCE.md) 了解测试和升级过程。

## 许可证

沿用上游仓库的许可证及第三方组件各自的许可证。WebClient v2 复用本仓库已有的开源网页连接代码。

## 开发分支中的扩展功能

当前开发分支增加了附加群组与所有用户共享地址簿、官方客户端 `--assign`、`--deploy` 接口、可选设备登记管理、设备与用户及设备组三级策略、活动连接管理和独立 API 管理令牌。这些功能仍处于里程碑开发阶段，正式版本保持原来的发布方式和版本编号。

- [功能完成情况](docs/FEATURE_MATRIX.md)
- [API 使用说明](docs/API.md)
- [实施台账](docs/IMPLEMENTATION_LEDGER.md)
- [接手开发指南](docs/HANDOFF.md)
- [测试与升级说明](docs/TESTING.md)

## 邮件与地址簿增强

此开发分支支持地址簿 CSV 导入导出和回收站恢复、SMTP 邮件服务、用户密码恢复以及成员邮件邀请。相关页面包括“地址簿”“地址簿回收站”“SMTP 邮件服务”和“成员邮件邀请”。使用 SMTP 密码时，需要设置 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`，完整配置参见 [部署与配置](docs/DEPLOYMENT.md)。

## 连接记录备注

管理员可在“全部连接记录”中为历史远程连接添加备注，记录用途和处理过程。连接记录导出 CSV 时也包含备注，方便日常归档。

## 双重认证

个人设置中支持可选的 TOTP 动态验证码和六个一次性恢复码。启用后，官方 RustDesk 客户端与 Web 管理控制台的密码登录都需要进行双重认证。服务器需要配置持久化的 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`，详情参见 [升级说明](docs/UPGRADE.md)。

## 邮箱验证码登录

可选启用邮箱登录验证。管理员配置 SMTP 后，用户可以在“个人设置”中完成邮箱绑定，后续密码登录会收到一次性验证码。若同时启用了 TOTP，将优先使用身份验证器。新安装默认不需要邮箱验证码。

## 多中继服务器

开发分支支持在 Web 管理控制台配置多个中继节点，并为每个节点设置优先使用的 IP 网络范围。`hbbs` 可以从 API 定期同步配置，在可用中继之间进行自动选择，并在节点断开连接时选择其他可用节点。具体设置参见 [部署说明](docs/DEPLOYMENT.md)。

## GeoLite2 国家级中继选择

为中继节点添加 `JP`、`US` 等国家代码后，hbbs 可以结合本地 GeoLite2 Country CSV，自动判断客户端 IP 所属国家并选择合适的中继节点。精确 CIDR 范围仍具有更高优先级。参见 [部署说明](docs/DEPLOYMENT.md)。

## 远程控制角色

开发分支支持官方 RustDesk 客户端的 14 项会话控制权限，可为设备、用户或设备组分配控制角色。管理员通过网页选择每项操作的“沿用设备设置”“禁止”或“允许”。启用时，hbbs 会在连接协商时向客户端传送生效角色。配置说明见 [部署文档](docs/DEPLOYMENT.md)。

## 管理员分级角色

可为团队成员分配多个管理角色，并逐项选择允许操作的后台功能。指定用户或用户组的角色可以限制用户与设备目录的查看范围。现有系统管理员拥有全部管理权限，普通成员默认不拥有后台管理权限。

管理页面新增“管理员角色”和“管理员角色分配”，详情参见 [API 说明](docs/API.md)。

## 可选的服务器连接访问规则

管理员可以通过“连接访问规则”设置哪些成员和用户组允许连接哪些设备。hbbs 可在普通连接和中继连接开始时请求 API 检查登录会话与设备访问权限。默认不启用，原有客户端可以继续使用。启用方法参见 [部署说明](docs/DEPLOYMENT.md)。


## 多平台客户端安装文件与自动化管理

开发分支的管理控制台提供“客户端安装包构建”，可保存服务器地址、公钥和 API 地址，创建 Windows、Linux、macOS 的 x64/ARM64 安装文件任务，并下载经过 SHA-256 检查的官方 RustDesk 1.5.1 安装程序。Windows 文件名包含服务器配置；Linux 和 macOS 提供带有配置脚本的 ZIP，保留原始 DEB 或 DMG 内容。详见 [客户端构建器说明](docs/CLIENT_BUILDER.md)。

管理员还可以创建独立 API 令牌、管理远程控制权限、编辑设备安全策略、批量维护共享关系及设备登记，并通过“邮件事件通知”选择需要提醒的操作。完整接口和启用方法见 [API 文档](docs/API.md) 及 [部署说明](docs/DEPLOYMENT.md)。

本分支的数据库迁移版本是 282。Linux、macOS 在真实设备上的安装、远程连接及正式 SMTP 服务的检查，继续以 [实施台账](docs/IMPLEMENTATION_LEDGER.md) 为准。正式版本发布需要单独确认。

数据库迁移、备份和恢复方法见 [数据库备份恢复说明](docs/DATABASE_BACKUP_RESTORE.md)。


## 个人事件推送与 GeoIP

用户可在控制台的“消息推送”页面添加 MessagePusher、SMTP 邮件和 Server酱渠道，并分别选择订阅事件。设备、连接和审计事件由后台异步发送。管理页面显示的 IP 地址将附注本地 GeoIP 查询结果。部署设置、事件说明及接口参见 [消息推送与 IP 地理位置](docs/PUSH_NOTIFICATIONS.md)。
