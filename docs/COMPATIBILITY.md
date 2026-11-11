# 版本与兼容性

本文区分 GitHub 已发行版本与正在开发的代码。数据库迁移版本应以实际运行程序中的 `cmd/apimain.go` 为准。

| 项目 | 正式发布版本 | 说明 |
| --- | --- | --- |
| RustDesk API | [v3.0.0-api.106](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.106) | 2026 年 11 月 10 日发布，数据库迁移版本 288 |
| RustDesk Server | [v1.1.17-api.21](https://github.com/HSOS6Miyabi/rustdesk-server/releases/tag/v1.1.17-api.21) | 配套维护的自托管 ID 与中继服务 |
| RustDesk 官方桌面客户端 | 1.5.1 | 本项目客户端构建器和主要协议测试使用的版本 |
| 当前 API 整改分支 | `feat/settings-center-reliability-20261111` | 开发版本，数据库迁移版本 290；还未创建正式 Release |

API 可与配套 Server 分开部署。Server 的 `v1.1.17-api.21` 组合镜像使用早期已发布的 API `v3.0.0-api.103`，因此升级 API 时需查看实际容器使用的 API 版本。多个镜像一起使用时，还要确认内部密钥、JWT 设置和运行参数一致。

## 数据库兼容性

项目支持 SQLite、MySQL 与 PostgreSQL。默认 SQLite 数据库位于 `data/rustdeskapi.db`。正式升级前应备份现有数据库，在独立环境恢复并测试。新增数据库字段不会自动出现在旧程序中；从开发版本回退至正式版本前，应先核对迁移表和业务数据，必要时恢复升级前的数据库备份。

## 功能启用条件

设备部署审批、连接访问规则、控制角色、国家级中继选择和多实例通知任务分别需要额外的设置。未启用时按原有自托管服务方式运行。详见 [部署说明](DEPLOYMENT.md)、[配置参考](CONFIG_REFERENCE.md) 与 [功能改动](FORK_CHANGELOG.md)。

发布之前必须完成 API、Server、真实客户端和数据库兼容检查，并在发行说明中记录实际验证的版本组合。