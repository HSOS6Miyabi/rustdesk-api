# 项目维护

## 源代码

API 仓库：https://github.com/HSOS6Miyabi/rustdesk-api

配套服务端：https://github.com/HSOS6Miyabi/rustdesk-server

API 原作者：lejianwen。服务端同步官方 RustDesk Server 的更新，同时保留与 API 联合使用的自定义功能。

## 日常开发

对 API 的修改需要检查 Go 后端、管理界面和 WebClient v2。管理界面位于 `resources/admin`，WebClient v2 位于 `resources/web2`。两个目录均随程序发布。

进行功能修改后，执行：

```bash
go test ./... -count=1
go build -o /tmp/rustdesk-api ./cmd/apimain.go
node --check resources/admin/app.js
node --test resources/admin/app.test.cjs
node --check resources/web2/app.js
cd resources/web/js && npm ci && npm run build
```

Redis 集成测试需要设置 `REDIS_TEST_ADDR`，例如 `127.0.0.1:6379`。GitHub Actions 会为测试启动 Redis 服务。

需要重点检查的行为包括：公共锚点组管理员的客户端可见范围、联系人单向共享、同一设备的共享来源合并、群组成员批量更新、个人群组的所有权校验、个人连接及文件传输日志的访问范围、连接双方的设备信息、OIDC 自动入组、地址簿分类数据保留，以及服务器指令超时。

## 发布

普通提交与标签推送仅运行验证工作流。常规里程碑发布在 `v*` 标签上手动启动 GitHub Actions，并启用 `milestone_release`。当前批准的正式版本为 `v3.0.0-api.110`；以后普通提交仍只运行检查，新的发行版本需要再次批准。

发布前检查配置文件中的端口及 JWT 密钥设置，并备份正式环境数据库。


## RustDesk 1.5.x 的自动化测试

连接审计测试应包括连接认证方式、双重认证、文件传输的连接编号、告警类型、大整数会话编号和重复提交处理。设备策略测试应包括 UUID 检查、策略分配、修改、撤销及自动心跳响应。

修改 WebClient 后，在 `resources/web/js` 目录执行：

```bash
npm ci
npm run build
node --test tests/webrtc.test.cjs tests/websock.test.cjs
```

RustDesk Server 的测试与编译：

```bash
cargo test --locked
cargo build --locked --release
```

GitHub Actions 在里程碑发布前检查 Go、JavaScript、浏览器和 Rust 功能，正式发布时才构建发行镜像并更新 GHCR 的 `latest` 标签。组合服务端镜像应选用已经发布的 API 版本。

浏览器界面和 WebRTC 通信需要使用 Chromium。安装测试工具后执行：

```bash
npm --prefix resources/admin install --no-save --no-package-lock --no-audit --no-fund playwright@1.56.0
resources/admin/node_modules/.bin/playwright install --with-deps chromium
node --test resources/admin/browser.e2e.cjs resources/admin/browser.webrtc.e2e.cjs
```

WebClient 同时支持 WebSocket 中继和 WebRTC。WebSocket 收到消息后直接通知等待中的操作；空闲连接保持活动状态。WebRTC 需要验证被控设备签名的 DTLS 证书信息。服务器的网络测试位于 `scripts/test_runtime.py` 和 `scripts/test_webrtc_rendezvous.py`。


## 2026 年 11 月里程碑发行

API 正式版本已更新至 `v3.0.0-api.110`。安装文件和多架构 Docker 镜像可从对应 GitHub Release 与容器仓库获取。

Server `v1.1.17-api.21` 的组合 Docker 镜像使用 API `v3.0.0-api.103`，升级时应检查容器内实际使用的 API 版本。Server 的对应版本也已发布，两个仓库的发行工作流和镜像构建检查均已完成。

已有 Go、浏览器、WebRTC、数据库和客户端自动化测试均应通过。两台实际设备的远程控制、不同系统的账户登录、真实 SMTP、跨地区公网中继及业务数据库恢复仍需具备相应环境后检查，具体内容保留在 [逐项任务记录](TASK_VERIFICATION_RECORD.md)。

## 设置中心与开发分支

设置中心与通知事件可靠性改进已合并至 `master`，从正式版本 `v3.0.0-api.107` 开始提供；数据库迁移版本为 **290**。

新功能需要执行真实 API 启动和关闭检查：

```bash
go build -o /tmp/rustdesk-api ./cmd/apimain.go
bash scripts/test_settings_startup.sh /tmp/rustdesk-api
```

使用说明见 [设置中心](SETTINGS_CENTER.md)、[上游同步](UPSTREAM_SYNC.md) 和 [版本兼容性](COMPATIBILITY.md)。

## v3.0.0-api.107 发行检查记录

2026 年 11 月 11 日，API 的 [PR #11](https://github.com/HSOS6Miyabi/rustdesk-api/pull/11) 已合并至 `master`，数据库迁移版本为 **290**。[GitHub Actions #42978795619](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42978795619) 的九项任务全部通过，涵盖 Go、浏览器、Linux AMD64/ARM64、Windows AMD64、正式 Release 和 Docker 镜像发布。

- 正式安装文件：[v3.0.0-api.107 Release](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.107)。
- Docker：`ghcr.io/hsos6miyabi/rustdesk-api:v3.0.0-api.107` 与 `latest` 均已发布，支持 AMD64 和 ARM64。
- 配套 Server 版本保持 `v1.1.17-api.21`，组合 Docker 镜像中现有 API 版本需要单独核查。
- 实际 SMTP 服务商、两台设备的远程操作、跨地区中继和生产数据恢复，应根据部署环境继续检查。

详细任务结果见 [统一实施台账](IMPLEMENTATION_LEDGER.md)、[逐项测试记录](TASK_VERIFICATION_RECORD.md) 与 [本次版本说明](releases/v3.0.0-api.107.md)。

## 2026 年 11 月 12 日：客户端安装文件生成

[PR #16](https://github.com/HSOS6Miyabi/rustdesk-api/pull/16) 为安装包构建器增加官方多版本选择与浏览器本地文件生成。浏览器选择本地 EXE、DEB、DMG 时，原始文件不会上传 API，也不计算 SHA-256。ZIP 使用 CRC-32。核查时运行 `service/client_release_test.go`、`resources/admin/client-local.test.cjs` 以及管理控制台 Chromium 测试。正式 API 版本 `v3.0.0-api.110` 继续使用数据库迁移版本 **290**。

## 2026 年 11 月 12 日：v3.0.0-api.110 正式发行

[发行工作流 #43157085657](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/43157085657) 的九项任务全部通过，已发布 Linux AMD64、Linux ARM64、Windows AMD64 安装文件及 AMD64/ARM64 Docker 镜像。

- [GitHub Release v3.0.0-api.110](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.110)
- Docker 镜像：`ghcr.io/hsos6miyabi/rustdesk-api:v3.0.0-api.110`
- 多架构镜像摘要：`sha256:85ab5519c8e41b3969dc8231e9fc693626d50ea3f94bb1d10fafddf150cc9f42`
- 数据库迁移版本：**290**

本地文件方式由浏览器直接生成，文件不上传 API，也不计算 SHA-256。具体操作和适用平台见 [客户端安装文件制作说明](CLIENT_BUILDER.md)。
