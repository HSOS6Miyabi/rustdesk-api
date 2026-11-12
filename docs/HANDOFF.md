# 开发交接与项目维护

本项目基于 [lejianwen/rustdesk-api](https://github.com/lejianwen/rustdesk-api) 开发，并与 [HSOS6Miyabi/rustdesk-server](https://github.com/HSOS6Miyabi/rustdesk-server) 联合使用。API 的主分支为 `master`。最新正式 API 版本为 `v3.0.0-api.110`，配套 RustDesk Server 正式版本为 `v1.1.17-api.21`。设置中心与运行可靠性改进已通过 API [拉取请求 #11](https://github.com/HSOS6Miyabi/rustdesk-api/pull/11) 合并到 `master`。

## 接手工作顺序

1. 阅读 [功能差异](FORK_CHANGELOG.md)、[兼容版本](COMPATIBILITY.md) 与 [功能清单](FEATURE_MATRIX.md)，了解本项目相对于上游的扩展内容。
2. 分别克隆 API 与 Server，查看各自主分支最新提交以及未合并的拉取请求，保留正在进行的修改。
3. 阅读 [部署说明](DEPLOYMENT.md)、[配置参考](CONFIG_REFERENCE.md)、[设置中心](SETTINGS_CENTER.md) 与 [备份恢复](DATABASE_BACKUP_RESTORE.md)。
4. 运行 Go、JavaScript、Chromium、数据库升级与恢复测试，并在实际客户端环境检查需要 Server 联合工作的功能。
5. 合并上游时按照 [上游更新步骤](UPSTREAM_SYNC.md) 创建单独的维护分支，避免覆盖本项目增加的功能。

## 代码位置

| 目录 | 用途 |
| --- | --- |
| `cmd/apimain.go` | 启动、数据库迁移与后台任务初始化 |
| `config/` | 配置文件、环境变量及网页覆盖的配置定义 |
| `model/` | 用户、设备、群组、策略、通知及设置记录的数据结构 |
| `service/` | 核心业务逻辑、权限判断、缓存与通知发送 |
| `http/controller/` | 管理接口、客户端接口和内部服务接口 |
| `http/router/` | HTTP 路由与权限中间件的组合 |
| `resources/admin/` | 桌面、平板与手机管理控制台 |
| `resources/web2/`、`resources/web/js/` | WebClient v2、WebRTC 与 WebSocket |
| `docs/` | 功能、配置、部署、备份和维护说明 |
| `scripts/`、`.github/workflows/` | 构建、测试与发行检查 |

## 现有功能及启用条件

系统提供账户和设备管理、用户群组、联系人单向共享、公共成员、地址簿共享、CSV 导入导出、回收站、审计记录、设备策略、管理员委派权限、TOTP 与邮箱验证码、SMTP 邮件、密码恢复、成员邀请、个人事件推送、客户端安装文件构建与多中继管理。各功能的页面和主要代码见 [功能差异](FORK_CHANGELOG.md)。

严格设备登记、服务器执行的连接许可、控制角色以及多中继自动选择需要配套 Server 启用相应接口。未启用的高级功能沿用默认连接方式。上线和离线通知、共享事件任务均采用按需处理方式。

## 数据库升级与备份

正式 API `v3.0.0-api.110` 对应迁移版本 **290**。本次更新增加设置覆盖数据库表、修改记录及事件处理权标识。升级之前应备份数据库、`data/settings-bootstrap.json`、API 配置、Server 身份密钥和用于解密持久化凭据的密钥。

默认 SQLite 数据库位于 `data/rustdeskapi.db`。MySQL 与 PostgreSQL 使用各自的备份工具。先在独立环境恢复备份，运行升级和业务测试，再升级正式服务。

## 设置中心与多实例

系统管理员在 `/_admin/` 的“系统管理 → 设置中心”查看配置用途、当前运行值、来源、覆盖状态和修改记录。网页覆盖值高于环境变量，保存后重新启动 API 生效。数据库连接、网络监听、密钥及缓存类型等保存在本机 `data/settings-bootstrap.json`，一般应用设置保存在数据库。

多个 API 实例共享数据库时，每台机器仍需保存自己的启动文件，并在修改配置后分别重新启动。缓存较短时间内可能使用以前的策略状态，涉及安全规则的变更应一并检查配套 Server 的配置。

## 本地测试

```bash
go test ./... -count=1
go test -race ./service ./lib/cache ./http/... -count=1
go build -o /tmp/rustdesk-api ./cmd/apimain.go
node --check resources/admin/app.js
node --test resources/admin/app.test.cjs
bash scripts/test_settings_startup.sh /tmp/rustdesk-api
```

WebClient 编译、真实客户端协议和 Chromium 浏览器测试见 [测试说明](TESTING.md)。GitHub Actions 也会执行 SQLite、MySQL、PostgreSQL 迁移检查和 Docker 构建测试。

## 版本与发布

普通代码提交与拉取请求仅执行自动化检查。正式 Release 和 Docker 镜像发布必须先完成相应测试并取得明确批准。发布记录应写明 API、Server、客户端和数据库的兼容版本。

早期的开发记录见 [2026 年 11 月 9 日交接快照](HANDOFF_SNAPSHOT_2026-11-09.md)、[实施台账](IMPLEMENTATION_LEDGER.md) 与 [逐项检查记录](TASK_VERIFICATION_RECORD.md)。这些材料记录对应日期的开发和测试状态，当前状态以 GitHub 最新提交、拉取请求和发行标签为准。

## 2026 年 11 月 11 日发行记录

API 当日发行的版本为 [v3.0.0-api.107](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.107)，数据库迁移版本为 **290**。合并记录为 [PR #11](https://github.com/HSOS6Miyabi/rustdesk-api/pull/11)，正式发行工作流 [#42978795619](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42978795619) 的九项任务已通过。

当前可直接从 Release 下载三个平台的 API 安装文件。Docker 镜像 `ghcr.io/hsos6miyabi/rustdesk-api:v3.0.0-api.107` 和 `latest` 已完成 AMD64/ARM64 发布。详细功能与任务结果在 [实施台账](IMPLEMENTATION_LEDGER.md) 和 [逐项记录](TASK_VERIFICATION_RECORD.md) 中维护。

## 2026 年 11 月 12 日：安装包构建器

[API `v3.0.0-api.110`](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.110) 包含 [PR #16](https://github.com/HSOS6Miyabi/rustdesk-api/pull/16)。管理员可以选择官方 RustDesk 版本，或自行在浏览器中选择本地 EXE、DEB、DMG 文件生成安装配置包。浏览器不会将原始文件上传到服务器，也不计算 SHA-256。数据库迁移版本保持 **290**。详见 [客户端安装文件制作说明](CLIENT_BUILDER.md)。
