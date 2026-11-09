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

普通提交与标签推送仅运行验证工作流。常规里程碑发布在 `v*` 标签上手动启动 GitHub Actions，并启用 `milestone_release`。2026 年 11 月本次发行已获得维护者批准，合并到 `master` 时使用专用发行标识触发一次正式发布；以后的普通提交不会自动发布。

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

本次发行版本为 `v3.0.0-api.103`，使用 GitHub Actions 构建 Linux AMD64、Linux ARM64 与 Windows AMD64 安装文件，创建 GitHub Release，并发布 `ghcr.io/hsos6miyabi/rustdesk-api:v3.0.0-api.103` 和 `latest` 多架构 Docker 镜像。只有经维护者批准、包含专用发行标识的合并提交才会启动本次自动发布。

Server `v1.1.17-api.21` 的组合 Docker 镜像引用本次 API 镜像。应先检查 API 的 Release、镜像及两个平台的容器运行状态，再发布 Server 的相应版本。

已有 Go、浏览器、WebRTC、数据库和客户端自动化测试均应通过。两台实际设备的远程控制、不同系统的账户登录、真实 SMTP、跨地区公网中继及业务数据库恢复仍需具备相应环境后检查，具体内容保留在 [逐项任务记录](TASK_VERIFICATION_RECORD.md)。
