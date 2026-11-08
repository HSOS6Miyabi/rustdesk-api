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

## 2026-11-09：设备管理完善、三级策略、活动连接与 API 令牌

此阶段继续使用两个仓库的 `development/m1-device-management` 分支，所有修改通过草稿 PR 检查。以下是本轮新增和扩展功能的具体提交。

| 仓库 | 提交 | 功能 |
| --- | --- | --- |
| API | [47a71c2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/47a71c288a6dfae9028d7216b978ebc4fc0ce675) | 完善部署管理路由和 SQLite 集成测试 |
| API | [97e3ef3](https://github.com/HSOS6Miyabi/rustdesk-api/commit/97e3ef3235f4f5c89df24cb9e1637126cd2c30e9) | 可撤销和恢复设备部署许可，并检查对应设备身份 |
| API | [79c2d9d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/79c2d9d7fbc5617ede0e7dfd416a34c057a3c08d) | API 与 Server 支持共同的设备部署环境变量 |
| API | [bdf646a](https://github.com/HSOS6Miyabi/rustdesk-api/commit/bdf646ae81f76ce62edaffe87faddf9e5e899a56) | 加入设备、用户、设备组三层策略和后台管理 |
| API | [2a64af3](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2a64af37b3ba92f0650f744ece62428acaf286da) | 验证策略优先级、禁用处理、撤销和数据清理 |
| API | [722c595](https://github.com/HSOS6Miyabi/rustdesk-api/commit/722c5958f50949b2faaa7fe237b96b6711f100d8) | 记录客户端活动会话并通过心跳发送结束指令 |
| API | [ff2facd](https://github.com/HSOS6Miyabi/rustdesk-api/commit/ff2facdec81e87a0f20bec769169ccc9245f4c3d) | 活动连接与结束连接记录管理页面 |
| API | [bb7a1c9](https://github.com/HSOS6Miyabi/rustdesk-api/commit/bb7a1c9ff54956e437c1c8d62afbd25cf6db405b) | 会话心跳、指令执行和管理页面自动化测试 |
| API | [0c570fb](https://github.com/HSOS6Miyabi/rustdesk-api/commit/0c570fb77ca6daf1599659993ff9dba1ac396ba4) | 独立 API 管理令牌、权限校验和自动化设备接口 |
| API | [c6135a1](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c6135a151a22ff3bc55dab38b00cd26b6a67da9a) | 管理控制台的令牌创建、查看与撤销 |
| API | [81a656b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/81a656be715b8c9c4e31b53b134dfb1c9d45fd93) | API 令牌权限与官方设备命令集成测试 |
| API | [468bddf](https://github.com/HSOS6Miyabi/rustdesk-api/commit/468bddfbb90521d559cea0a31a8aadc79cf372d7) | 改善客户端认证组件的初始化检查 |
| API | [5310011](https://github.com/HSOS6Miyabi/rustdesk-api/commit/53100114da346d9d544a8686f9340920bebe763c) | 调整令牌创建后的即时显示 |
| API | [c4b1302](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c4b130274e80910878e3992859103eb77cac03b6) | 令牌创建页面独立处理保存操作 |
| Server | [8995074](https://github.com/HSOS6Miyabi/rustdesk-server/commit/89950741e6c156d6ed7cb24c348a97441f3c0ce0) | 增加服务端实施台账与接手文档 |
| Server | [cd67739](https://github.com/HSOS6Miyabi/rustdesk-server/commit/cd6773901134fd38ca566af73de906e014257637) | 真实 `hbbs` 设备登记通信测试 |
| Server | [ec48c9b](https://github.com/HSOS6Miyabi/rustdesk-server/commit/ec48c9bb5414d93d9b486cc158cbc42216fb8199) | 设备恢复连接时重新检查登记许可 |
| Server | [bfd1d8e](https://github.com/HSOS6Miyabi/rustdesk-server/commit/bfd1d8e2ca3f7d513e7686f42ead026826606b95) | 仅接受 API 明确启用后的严格登记许可 |
| Server | [902293d](https://github.com/HSOS6Miyabi/rustdesk-server/commit/902293d49f17d831922237718ff389b3a1178267) | 完善服务端环境变量、测试说明和接手文档 |

### 数据变更与兼容性

新增 `DevicePolicyUserAssignment`、`DevicePolicyGroupAssignment`、`DeviceActiveConnection`、`DeviceDisconnectRequest` 和 `APIToken`，并为 `DevicePolicy` 增加启停状态。旧策略保持有效。

活动连接使用客户端 `/api/heartbeat` 中的 `conns` 信息；结束指令使用 `disconnect`，对不支持报告活动连接的旧客户端不会显示虚假的活动会话。管理令牌采用 `rdapi_` 前缀，仅用于有相应权限的设备命令接口。

### 已确认的自动化测试

- [API 设备命令、部署管理与数据库测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42519664684)。
- [API 三级策略、回退规则及浏览器测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42520109512)。
- [API 活动连接、心跳指令及浏览器测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42520568742)。
- [Server 严格设备登记、通信协议和 Rust 测试通过](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42519807127)。
- [Server 文档更新后的自动化测试通过](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42521370716)。

API 管理令牌的最终浏览器测试及完整 Go 测试，应以[API 草稿 PR](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) 最新检查结果为准。

### 后续功能

商业版仍待实现的功能以 [FEATURE_MATRIX.md](FEATURE_MATRIX.md) 为准。继续开发时，更新本台账、对应提交编号、测试结果和 [HANDOFF.md](HANDOFF.md)。
