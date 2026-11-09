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

## 2026-11-09：共享地址簿的附加群组与所有用户共享

这一组修改主要帮助小型团队快速共享常用设备列表。新增功能保留现有个人群组、共享组及公共锚点组的行为。

| 仓库 | 提交 | 完成内容 |
| --- | --- | --- |
| API | [ef495dc](https://github.com/HSOS6Miyabi/rustdesk-api/commit/ef495dc3fea1a615dc152200e9dac64da4ee9ba7) | 支持附加群组和所有用户共享；客户端共享地址簿列表使用实际权限 |
| API | [5938de3](https://github.com/HSOS6Miyabi/rustdesk-api/commit/5938de324923bbc2e5e0ddbff0a412f8e4c7f243) | 覆盖多群组共享、权限优先级、所有用户规则修改及管理页面测试 |
| API | [bafa58d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/bafa58da217efab387eb3db4b46e7f8f1f746f5c) | 验证原生客户端 `/api/ab/shared/profiles` 在多种权限组合下的响应 |

### 主要修改

- `service/addressBook.go`：读取个人、所有群组及公开共享规则，计算具体用户、群组和公开规则的优先级。
- `model/addressBook.go`：共享对象新增“所有用户”，使用 `type=3, to_id=0`。
- `http/controller/api/ab.go`：返回每个地址簿的最终权限，保留公共锚点组原有的可见范围。
- `http/controller/admin/addressBookCollectionRule.go`、`http/controller/admin/my/addressBookCollectionRule.go`：管理规则时支持所有用户。
- `resources/admin/app.js`：个人共享规则页面及所有用户选项。

无需迁移已有共享规则或改变原有设备 ID。新增的共享类型使用现有数据表字段。

### 自动化测试

- [共享功能实现后的检查通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42522339510)。
- [多群组和公开规则测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42522474127)。
- [原生共享地址簿接口集成测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42522571693)。

详见 [API 使用说明](API.md)、[测试说明](TESTING.md) 及 [功能清单](FEATURE_MATRIX.md)。后续还可补充更多第三方地址簿导入格式和批量共享功能。

## 2026-11-09：地址簿导入导出、恢复、SMTP 与账户邮件功能

本次继续开发以小型团队日常使用为重点，所有程序均保留在 `development/m1-device-management`，两个草稿 PR 均不发布发行版。

### 已提交的功能

| 仓库 | 提交 | 内容 |
| --- | --- | --- |
| API | [a002498](https://github.com/HSOS6Miyabi/rustdesk-api/commit/a0024985c0bb07e3627a25f9a56e7455addf45b1) | 地址簿 CSV 导入导出，按用户限制访问，最多 5000 条、4 MB，导入事务处理 |
| API | [d30ad8c](https://github.com/HSOS6Miyabi/rustdesk-api/commit/d30ad8c9cc91aea2f3172aa48f41bf7c1374a8b0) | CSV 导入导出、中文内容、重复记录与跨用户访问的测试 |
| API | [2a43f80](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2a43f80d1db8c06282aa4a10f8b5b700b2edd804) | 管理控制台 CSV 导入导出按钮及浏览器操作 |
| API | [c07cc6f](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c07cc6f2df867f4aaddf84b146b2806bcbf4780d) | 地址簿分类编号导入测试 |
| API | [21f70c7](https://github.com/HSOS6Miyabi/rustdesk-api/commit/21f70c7977d4c9ae1fea9f2f7535d41053fcd282) | 地址簿条目回收站与删除整个分类时的记录处理 |
| API | [918089c](https://github.com/HSOS6Miyabi/rustdesk-api/commit/918089c83023bd1023deb114efcbb3df764e189e) | 恢复、彻底删除、查询回收站接口及账户删除关联处理 |
| API | [6e13012](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6e130123e3d06d524cc7accd78592ca20a7cc95c) | 回收站管理界面与浏览器测试 |
| API | [6885810](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6885810b0643aa5972c498f1aa9a2636e32e80b7) | 恢复设备配置、归属验证、重名检查及分类删除的测试 |
| API | [d7997c0](https://github.com/HSOS6Miyabi/rustdesk-api/commit/d7997c012dcf5287e40a6b4e53bd8736279b1f3b) | SMTP 服务器配置、密码加密存储、发送邮件与测试接口 |
| API | [df0f509](https://github.com/HSOS6Miyabi/rustdesk-api/commit/df0f509a01ca9ce60282dab31ddf96dee4c817cf) | 模拟 SMTP 服务器发送测试、加密保存和配置修改 |
| API | [1a06af2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/1a06af26d5c28fa2b24700c939fcb90c8fca6ab0) | SMTP 管理页面及发送测试邮件操作 |
| API | [78358d8](https://github.com/HSOS6Miyabi/rustdesk-api/commit/78358d887f827537e980b5607af55743b2a53f92) | 密码恢复请求、15 分钟一次性链接、旧会话撤销 |
| API | [6eb9ee6](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6eb9ee61b7650afbf92d0f9e3455b755969914cb) | 邮件恢复完整服务测试与账户状态处理 |
| API | [200cfad](https://github.com/HSOS6Miyabi/rustdesk-api/commit/200cfad69ec2cdd1b6b781ce46990a39ffe48a8c) | 网页登录入口和密码恢复表单 |
| API | [a922f32](https://github.com/HSOS6Miyabi/rustdesk-api/commit/a922f3235e537dd82cd2bdfc4e796c357277b55f) | 登录页面与恢复表单浏览器测试 |
| API | [2a42e2d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2a42e2d4bd1f66470e936433c54a181025b6679d) | 团队邮箱邀请、按指定群组注册和邀请撤销 |
| API | [c482a4b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c482a4b4d29cc366031ac712d325db558db98c0f) | 邀请邮件与注册、失效、撤销的自动测试 |
| API | [0a37914](https://github.com/HSOS6Miyabi/rustdesk-api/commit/0a37914e573fa8d2b5e7a3048e0b7da0a0858316) | 管理员邀请页面、登录页面处理邮件链接 |
| API | [453ce53](https://github.com/HSOS6Miyabi/rustdesk-api/commit/453ce536e146dd1f9ac9179638409a23d6262848) | 邀请发送、撤销及注册页面的浏览器自动化测试 |

### 数据库及部署

地址簿条目新增 `deleted_at`，普通删除后可从回收站恢复。删除整个地址簿分类时，其关联的活动条目和已删除条目都会被彻底清除。

新增 `smtp_settings`、`password_reset_tokens`、`user_invitations`。已有用户、密码与群组保持原样。SMTP 邮箱凭据使用 `RUSTDESK_API_MAIL_ENCRYPTION_KEY` 加密保存。更换该密钥前，应在管理控制台重新配置 SMTP 密码。

密码恢复仅修改本地账户密码，并撤销该账户已有登录会话；OIDC 与 LDAP 的身份配置保持不变。邮件邀请创建的新账户会归属于指定用户组。

### 最新测试状态

- [地址簿 CSV 完整测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42558570753)，通过。
- [回收站恢复及管理界面测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42559071829)，通过。
- [SMTP 配置及邮件传送测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42559555373)，通过。
- [密码恢复服务测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42559793707)，通过。
- [邀请注册、邮件找回及管理页面的最新综合测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42560569953)，通过。

当前开发仓库对应 Server 开发提交仍是 [902293d](https://github.com/HSOS6Miyabi/rustdesk-server/commit/902293d49f17d831922237718ff389b3a1178267)。未创建里程碑版本标签或新的 Release。

## 2026-11-09：历史连接备注

新增管理员在已有连接记录中填写与修改备注的功能，备注会随连接日志一起查询和导出。原有连接 ID、会话信息与认证记录保持不变。

| 仓库 | 提交 | 具体内容 |
| --- | --- | --- |
| API | [507210f](https://github.com/HSOS6Miyabi/rustdesk-api/commit/507210f3183b534db3858be3ccdc6a82939c69d5) | 为连接审计记录增加备注、更新人编号、HTTP 接口与网页编辑功能 |
| API | [9664256](https://github.com/HSOS6Miyabi/rustdesk-api/commit/9664256840d3c89edc188abbf729f77cc7b59413) | 测试备注保存、清除、非法长度、CSV 导出和 Chromium 页面操作 |

数据库 `audit_conns` 增加 `note`、`note_updated_by`；使用现有数据库迁移方式，旧连接记录保持可用。

管理员通过 `POST /api/admin/audit_conn/note` 修改备注，最多 2000 个字符。管理操作自动记录在原有操作日志中。备注同样包含在 `GET /api/admin/audit/export?kind=audit_conn` 的 CSV 文件中。

自动化测试见 [API 草稿 PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) 的最新提交及 [GitHub Actions](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42561282556)。这部分仅修改 API，无须更新 Server。

## 2026-11-09：TOTP 双重认证和一次性恢复码

本模块对小型团队默认关闭。用户可以通过个人设置开启身份验证器，登录时按官方客户端双重认证请求格式验证。支持 30 秒动态验证码、6 个一次性恢复码及验证码重复使用检查。

| 仓库 | 提交 | 内容 |
| --- | --- | --- |
| API | [6115846](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6115846009eed5c7e01a4d2b58ff01da4938193e) | TOTP 时间校验、绑定、恢复码、登录挑战及数据库迁移 |
| API | [6cfd42b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6cfd42bae6040d16c6eeae9af82a8b560313ac1f) | 官方客户端和网页密码登录的双重认证步骤 |
| API | [1299065](https://github.com/HSOS6Miyabi/rustdesk-api/commit/12990659d510794924d44ce2b8f46140f3d30707) | 个人设置中的双重认证启停、恢复码管理 |
| API | [2943406](https://github.com/HSOS6Miyabi/rustdesk-api/commit/29434069a2fa6e6d6d2d08c03ff62f2c9b903c86) | RFC 6238 标准验证码及绑定、恢复码与挑战限制测试 |
| API | [760fbb2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/760fbb25add2190ce75b626e649857622c19c425) | 官方 RustDesk 客户端登录协议的请求与响应测试 |
| API | [eefe3f2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/eefe3f2b97df1d5b6d7bfe386e448b3cc70df786) | 管理网页 TOTP 绑定、恢复码显示及登录浏览器测试 |
| API | [ec31834](https://github.com/HSOS6Miyabi/rustdesk-api/commit/ec3183425294169f22ec64938cab0365a0f41217) | 删除账户时删除 TOTP 数据、登录挑战及密码恢复申请；撤销该账户发出的待接受邀请 |

### 数据、接口及兼容性

增加 `user_totps`、`totp_recovery_codes`、`totp_login_challenges`，数据库迁移版本更新到 274。TOTP 密钥加密保存；恢复码只在生成时显示，数据库只保存摘要。

官方客户端登录首先返回 `type=email_check`、`tfa_type=tfa_check` 和一次性 `secret`；用户提交动态验证码后再创建实际登录令牌。网页通过 `/api/admin/mfa` 管理绑定状态。

目前复用邮件模块使用的 `RUSTDESK_API_MAIL_ENCRYPTION_KEY` 加密 TOTP 密钥。该环境变量在不使用 SMTP 的安装中也需要设置，开启双重认证前应保存并备份。

### 验证

- [TOTP 标准算法与恢复码测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42562369988)，通过。
- [RustDesk 官方客户端双重认证协议测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42562498241)，通过。
- [Web 控制台绑定、恢复码及登录浏览器测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42562596974)，通过。

真实 Windows、macOS、Linux、Android 客户端登录需要在完成里程碑前结合实际设备进行检查。
