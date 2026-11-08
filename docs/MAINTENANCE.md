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

推送版本标签，例如 `v3.0.0` 后，GitHub Actions 在 Go 测试、前端交互测试及 WebClient 连接组件构建通过后，发布可执行程序压缩包到 GitHub Release。主分支和版本标签还会构建 Docker 镜像，分别提供 amd64 与 arm64 架构。

发布前检查配置文件中的端口及 JWT 密钥设置，并备份正式环境数据库。


## RustDesk 1.5.x 的自动化测试

连接审计测试应包括连接认证方式、双重认证、文件传输的连接编号、告警类型、大整数会话编号和重复提交处理。设备策略测试应包括 UUID 检查、策略分配、修改、撤销及自动心跳响应。

修改 WebClient 后，在 `resources/web/js` 目录执行：

```bash
npm ci
npm run build
node --test tests/webrtc.test.cjs
```

RustDesk Server 的测试与编译：

```bash
cargo test --locked
cargo build --locked --release
```

GitHub Actions 在发布前执行 Go、JavaScript、浏览器和 Rust 测试，构建跨架构程序及 Docker 镜像，并更新 GHCR 的 `latest` 标签。组合服务端镜像应选用已经发布的 API 版本。
