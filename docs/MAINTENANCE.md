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
node --check resources/web2/app.js
```

Redis 集成测试需要设置 `REDIS_TEST_ADDR`，例如 `127.0.0.1:6379`。GitHub Actions 会为测试启动 Redis 服务。

需要重点检查的行为包括：公共锚点组管理员的客户端可见范围、联系人单向共享、同一设备的共享来源合并、群组成员批量更新、OIDC 自动入组、地址簿分类数据保留，以及服务器指令超时。

## 发布

推送版本标签，例如 `v3.0.0` 后，GitHub Actions 在 Go 测试和 JavaScript 语法检查通过后，发布可执行程序压缩包到 GitHub Release。主分支和版本标签还会构建 Docker 镜像，分别提供 amd64 与 arm64 架构。

发布前检查配置文件中的端口及 JWT 密钥设置，并备份正式环境数据库。
