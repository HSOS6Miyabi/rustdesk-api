# 开发实施台账

## 2026-11-09：第一个开发阶段开始

目标：兼容官方客户端设备分配与部署命令，建立可选设备登记检查，并保留现有小型团队默认注册方式。

仓库：
- API：`HSOS6Miyabi/rustdesk-api`，开发分支 `development/m1-device-management`
- Server：`HSOS6Miyabi/rustdesk-server`，开发分支 `development/m1-device-management`

### 发布流程

| 仓库 | 提交 | 更改 |
| --- | --- | --- |
| API | [b9da00b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/b9da00b71d3ff43b0c943a8817302b71e0548d3d) | 只有手动确认里程碑的标签工作流可以发布 |
| Server | [b4e1fb6](https://github.com/HSOS6Miyabi/rustdesk-server/commit/b4e1fb6ec93c74196cd91461b710176c07f0a8cc) | 仅允许手动确认后发布二进制程序和镜像 |

### 功能修改

| 仓库 | 提交 | 代码与行为 |
| --- | --- | --- |
| API | [edbeb5d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/edbeb5d374fe4c46084d9aaac742eab2a318a5ae) | 官方 `/api/devices/cli`、`/api/devices/deploy`；设备登记记录；内部登记检查；迁移和配置 |
| API | [0e66647](https://github.com/HSOS6Miyabi/rustdesk-api/commit/0e666477447fddbe6bcc7ef14d1f942b4d9f259a) | 后台部署记录、设备备注、命令行分配及登记接口测试 |
| Server | [33a6108](https://github.com/HSOS6Miyabi/rustdesk-server/commit/33a6108638d7cbc7bb240fd03768355a4bb6b105) | `RegisterPk` 可选 API 登记检查、`RequestRelay` 令牌检查、缺少 JWT 密钥时拒绝启动 |

### 配置、数据库及兼容性

API 新增 `app.require-device-deployment` 和 `app.internal-key`；数据库自动迁移 `DeviceDeployment` 表、设备和地址簿的 `note` 字段。

Server 新增 `REQUIRE_DEVICE_DEPLOYMENT`、`RUSTDESK_API_INTERNAL_KEY`、`RUSTDESK_API_INTERNAL_URL`。

严格登记默认关闭，API 的 `--deploy` 返回 `NOT_ENABLED`。启用时必须同时配置 API 和 Server，并使用相同的内部访问密钥。现有设备与地址簿原始记录不删除。

### 测试和后续检查

测试说明见 [TESTING.md](TESTING.md)。本阶段开发分支通过草稿 PR 执行 CI，实际通过情况以 PR 的检查结果为准。连接真实桌面、文件传输及严格设备部署还需要在具备客户端的环境验证。

待实施功能持续更新于 [FEATURE_MATRIX.md](FEATURE_MATRIX.md)；交接说明见 [HANDOFF.md](HANDOFF.md)。
