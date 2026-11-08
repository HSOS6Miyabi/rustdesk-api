# RustDesk API

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

提交代码和提交拉取请求时，GitHub Actions 会检查 Go、JavaScript 和浏览器功能。日常提交与推送标签不会自动发布。完成里程碑后，使用 `v*` 标签手动启动工作流并明确选择 `milestone_release`，才会创建 GitHub Release 和发行版 Docker 镜像。

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
