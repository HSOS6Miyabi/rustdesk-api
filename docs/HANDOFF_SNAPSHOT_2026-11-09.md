# RustDesk 两仓库开发交接快照

更新日期：2026 年 11 月 9 日。此文档是交接时的入口，后续开发者应先核对 GitHub 分支的当前提交和 CI，再按任务清单继续。详细记录见 [统一实施台账](IMPLEMENTATION_LEDGER.md) 和 [T001–T022 逐项测试记录](TASK_VERIFICATION_RECORD.md)。

## 一、两个仓库及发布状态

| 项目 | API | Server |
| --- | --- | --- |
| 仓库 | [rustdesk-api](https://github.com/HSOS6Miyabi/rustdesk-api) | [rustdesk-server](https://github.com/HSOS6Miyabi/rustdesk-server) |
| 开发分支 | `development/m1-device-management` | `development/m1-device-management` |
| 最后核实的功能提交 | [`dc0a998369d0d1cee8d9daba7a837b9a2ba966f0`](https://github.com/HSOS6Miyabi/rustdesk-api/commit/dc0a998369d0d1cee8d9daba7a837b9a2ba966f0) | [`eb4b586f9c65a1060dea8ce56b4809dec746c788`](https://github.com/HSOS6Miyabi/rustdesk-server/commit/eb4b586f9c65a1060dea8ce56b4809dec746c788) |
| 草稿拉取请求 | [#6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) | [#4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4) |
| 目标默认分支 | `master` | `forapi` |
| 原有正式版本 | `v3.0.0-api.96` | `v1.1.17-api.9` |

当前 API 数据库迁移版本为 **282**。开发分支中的代码未合并至默认分支，也未为本阶段创建正式 Release 或正式容器镜像。除非维护者明确批准首个里程碑发布，两个 PR 保持 Draft，禁止自动合并、打发行标签和上传正式镜像。

## 二、最近已确认成功的自动化检查

| 检查内容 | GitHub Actions |
| --- | --- |
| API Go、Chromium 管理页面、WebClient、WebRTC | [#42646358717](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42646358717) |
| 官方六种桌面安装文件 SHA-256、配置脚本 | [#42646358868](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42646358868) |
| 官方 Linux 1.5.1 客户端 `--deploy`、`--assign`，与实际 API 通信 | [#42646358750](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42646358750) |
| SQLite、MySQL 8.4、PostgreSQL 16 模拟升级及备份恢复 | [#42646358726](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42646358726) |
| Windows、Linux、macOS 官方客户端启动；Android 模拟器安装启动 | [#42646358634](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42646358634) |
| Server Rust 测试、真实 hbbs/hbbr、严格登记、JWT、访问许可与控制角色组合 | [#42647450900](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42647450900) |
| AMD64/ARM64 普通进程部署、组合容器、重启与数据保持 | [#42647450607](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42647450607) |

上述成功记录均对应本节的已核实功能提交。交接文档变动之后的提交需要另行查看新一轮检查。真实公网、多平台账户登录和两台设备间的实际远程控制仍需专门测试。

## 三、功能交付范围

统一台账登记 **F01–F20 共 20 个功能条目**。当前代码已包含账户、群组、联系人、地址簿、设备登记与管理、独立 API 令牌、管理员范围权限、客户端安全策略、控制角色、严格连接许可、多中继与 GeoLite2、WebClient、审计和 CSV、邮件事件通知、客户端安装文件制作及 Docker/普通进程部署。

客户端构建器支持 Windows、Linux、macOS 的 x64 与 ARM64 六种 RustDesk 1.5.1 官方文件。Windows 保留 EXE 文件字节并按官方方式生成带服务器信息的文件名；Linux/macOS 使用保留原始 DEB/DMG 的 ZIP 包，附配置脚本。Android 已在模拟器启动官方 APK。实际客户端登录、远程控制、正式服务器连接和真实系统安装细节仍需测试。

## 四、22 项工作状态及接手顺序

| 编号 | 优先级 | 当前结果与交接任务 |
| --- | --- | --- |
| T001 | P0 | 实际 API 与 hbbs 的登记、撤销、身份冲突检查通过；官方 Linux 1.5.1 客户端与实际 API 的部署和直接分配通过；双终端远程重连待检查 |
| T002 | P0 | 登录与双重认证代码测试通过；在 Windows/Linux/macOS/Android/iOS 可用环境检查密码、TOTP、邮箱、OIDC/LDAP 与恢复码 |
| T003 | P0 | 14 项控制角色信令检查通过；用两台真实设备验证各权限的允许、禁止及默认行为 |
| T004 | P0 | 实际 API 与 hbbs 的 JWT、连接权限、控制角色和中继请求组合在 AMD64/ARM64 通过；真实客户端的直连、中继及 WebRTC 远程会话待检查 |
| T005 | P0 | MMDB、Country CSV、节点失效和 reload-geo 测试通过；使用实际 GeoLite2 文件与两个地区的公网中继完成实测 |
| T006 | P0 | 三种临时数据库迁移与恢复检查通过；从正式业务环境获取受保护备份并在独立环境恢复 |
| T007 | P0 | 模拟 SMTP 及隐式 TLS、STARTTLS、身份认证测试通过；完成实际邮箱的邀请、找回密码、验证码与通知收件 |
| T008 | P0 | AMD64/ARM64 普通进程、Docker 与重启测试通过；在目标主机检查域名、证书、权限、数据目录及恢复 |
| T009 | P0 | Chromium 桌面、平板、手机操作及截图通过；使用真实管理账户检查完整页面与中文提示 |
| T010 | P0 | 官方 1.4.5/1.5.0/1.5.1 源代码字段检查、1.5.1 多系统启动和 Linux 客户端与实际 API 的设备命令通过；继续验证登录、双重认证及多版本远程连接 |
| T011 | P0 | 双架构编译和部署预演通过，未发行正式安装包或镜像 |
| T012 | P0 | 台账和草稿 PR 已整理；待所有 P0 实测结果和维护者批准后再决定是否发布 |
| T013 | P1 | 六种官方安装文件及任务管理通过测试；检查每种目标系统的实际安装、配置、签名和远程连接 |
| T014 | P1 | 安全策略心跳、更新和清除测试通过；在真实客户端检查策略是否生效 |
| T015 | P1 | 独立 API 令牌权限、有效期及撤销已实现，自动测试通过 |
| T016 | P1 | 管理员按用户及用户组范围操作已实现，自动测试通过 |
| T017 | P1 | 联系人、共享地址簿、部署许可和回收站批量处理已实现，自动测试通过 |
| T018 | P1 | GeoLite2 Country MMDB 与 CSV 已实现，自动测试通过；正式公网行为计入 T005 |
| T019 | P1 | 登录异常、设备部署、会话结束及文件传输通知已实现；真实收件计入 T007 |
| T020 | P1 | 设备、地址簿与审计筛选、CSV 导出、连接备注已实现；真实团队使用检查待执行 |
| T021 | P2 | 已整理官方 1.5.1 新功能候选；按新版本持续检查 |
| T022 | P2 | 中文页面和移动浏览器测试通过；其他语言及无障碍操作持续完善 |

接手建议先完成 **T001、T004、T002、T003**，随后是 **T007、T005**，再完成正式数据、部署及用户界面检查 **T006、T008、T009、T010、T013、T014、T020**。最终仅在 **T012** 获得明确批准后执行发行。每项的实际执行步骤、源文件、预期结果与完成标准见 [T001–T022 逐项测试记录](TASK_VERIFICATION_RECORD.md)。

## 五、仍需提供的实际环境

- 至少两台能互相远程连接的设备，及 Windows/Linux/macOS、Android 和可用的 iOS 客户端；测试账户需与正式业务账户分开。
- 测试环境的完整 API、hbbs、hbbr 访问地址与管理权限；严格模式需要一致的内部密钥和 JWT 设置。
- 至少两个不同地区的公网中继节点，以及有使用权限的 GeoLite2 Country MMDB 或 CSV。
- 可收件的测试邮箱及 SMTP 服务商，独立的 OIDC/LDAP 测试账号（如需测试这些登录方式）。
- 正式业务数据库的受保护备份和独立恢复环境；同时保存 Server 身份密钥、API 配置及客户端构建文件。
- 维护者针对正式版本和发行镜像的明确审批。

没有上述外部环境时，可继续执行单元、浏览器、临时数据库及本地通信测试；这些结果不得填写为真实生产环境测试通过。严禁把用户密码、设备密钥、数据库密码、邮箱密钥或原始令牌写入提交记录和截图。

## 六、接手人的操作步骤

1. 分别克隆两个仓库，检出 `development/m1-device-management`；Server 额外运行 `git submodule update --init --recursive`。
2. 核对两个最新提交与 [API 草稿 PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6)、[Server 草稿 PR #4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4) 的 CI 状态，不要只依赖本文列出的历史提交。
3. API 执行 `go test ./...`、`node --test resources/admin/app.test.cjs`，需要浏览器环境时执行 `node --test resources/admin/browser.e2e.cjs`；同时执行 `python3 scripts/verify_task_ledger.py`。
4. Server 执行 `cargo test --lib --tests`、`cargo build --bins`，并运行 `scripts/test_runtime.py`、`scripts/test_strict_combination.py` 和其他对应真实 hbbs 测试；跨架构组合检查使用 `.github/workflows/milestone-preflight.yml`。
5. 对真实环境项目逐项写入测试日期、设备和客户端版本、所用配置、预期与实际行为、CI 或截图地址。更新 [逐项测试记录](TASK_VERIFICATION_RECORD.md) 及 [实施台账](IMPLEMENTATION_LEDGER.md)。
6. 完成全部发布前检查后，将测试结果提交维护者审阅。在得到明确批准之前，两个 PR 保持草稿，不合并、不创建发行标签、不推送正式 Docker 镜像。

相关文档：[架构说明](ARCHITECTURE.md)、[部署说明](DEPLOYMENT.md)、[数据库备份恢复](DATABASE_BACKUP_RESTORE.md)、[客户端安装文件](CLIENT_BUILDER.md)、[API 接口](API.md)、[测试说明](TESTING.md)、[官方功能研究](UPSTREAM_FEATURE_WATCH.md)。
