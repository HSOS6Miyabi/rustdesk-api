# 版本与兼容性

本文区分 GitHub 已发行版本与正在开发的代码。数据库迁移版本应以实际运行程序中的 `cmd/apimain.go` 为准。

| 项目 | 正式发布版本 | 说明 |
| --- | --- | --- |
| RustDesk API | [v3.0.0-api.110](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.110) | 2026 年 11 月 12 日发行，数据库迁移版本 **290** |
| RustDesk Server | [v1.1.17-api.21](https://github.com/HSOS6Miyabi/rustdesk-server/releases/tag/v1.1.17-api.21) | 配套维护的自托管 ID 与中继服务 |
| RustDesk 官方桌面客户端 | 可选择多个正式版本 | 构建器从官方发行版选择对应平台的文件；已有协议测试主要使用 1.5.1 |
| 设置中心与性能改进 | [拉取请求 #11](https://github.com/HSOS6Miyabi/rustdesk-api/pull/11) | 包含在 API `v3.0.0-api.107` 中 |
| 控制台与 WebClient 改进 | [拉取请求 #13](https://github.com/HSOS6Miyabi/rustdesk-api/pull/13) | 包含在 API `v3.0.0-api.108` 中 |
| 个人数据与接口检查 | [拉取请求 #14](https://github.com/HSOS6Miyabi/rustdesk-api/pull/14) | 包含在 API `v3.0.0-api.109` 中 |
| 多版本与本地安装包生成 | [拉取请求 #16](https://github.com/HSOS6Miyabi/rustdesk-api/pull/16) | 包含在 API `v3.0.0-api.110` 中，支持浏览器本地处理 EXE、DEB、DMG |

API 可与配套 Server 分开部署。Server 的 `v1.1.17-api.21` 组合镜像使用早期已发布的 API `v3.0.0-api.103`，因此升级 API 时需查看实际容器使用的 API 版本。多个镜像一起使用时，还要确认内部密钥、JWT 设置和运行参数一致。

## 数据库兼容性

项目支持 SQLite、MySQL 与 PostgreSQL。默认 SQLite 数据库位于 `data/rustdeskapi.db`。正式升级前应备份现有数据库，在独立环境恢复并测试。新增数据库字段不会自动出现在旧程序中；回退至旧版本前，应先核对迁移表和业务数据，必要时恢复升级前的数据库备份。

## 功能启用条件

设备部署审批、连接访问规则、控制角色、国家级中继选择和多实例通知任务分别需要额外的设置。未启用时按原有自托管服务方式运行。详见 [部署说明](DEPLOYMENT.md)、[配置参考](CONFIG_REFERENCE.md) 与 [功能改动](FORK_CHANGELOG.md)。

本次版本已通过 API 自动化测试以及 SQLite、MySQL、PostgreSQL 的升级检查。正式 SMTP、真实客户端连接和生产数据恢复仍需在实际部署环境中检查；升级前应备份数据库与加密密钥。
## 发行测试记录

[正式发行工作流 #43157085657](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/43157085657) 的九项任务均已通过；其 Release 包含 Linux AMD64、Linux ARM64 和 Windows AMD64 文件，Docker 镜像支持 `linux/amd64` 与 `linux/arm64`。数据库的自动升级和恢复检查见 [#42975509940](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42975509940)。这些测试记录与实际业务数据恢复、真实设备连接和正式 SMTP 收发分别保存。
