# RustDesk 两仓库发布与维护说明

更新日期：2026 年 11 月 10 日。本文记录已发布版本、对应的代码和自动化测试，以及仍需实际环境检查的工作。详细结果见[统一实施台账](IMPLEMENTATION_LEDGER.md)和[T001–T022 逐项测试记录](TASK_VERIFICATION_RECORD.md)。

## 一、两个仓库及发布状态

| 项目 | API | Server |
| --- | --- | --- |
| 仓库 | [rustdesk-api](https://github.com/HSOS6Miyabi/rustdesk-api) | [rustdesk-server](https://github.com/HSOS6Miyabi/rustdesk-server) |
| 正式版本所在分支 | `master` | `forapi` |
| 已发行的源代码提交 | [`a6d338821cfba2daffa34f9cc3b06605ea1edf03`](https://github.com/HSOS6Miyabi/rustdesk-api/commit/a6d338821cfba2daffa34f9cc3b06605ea1edf03) | [`2d0896ecea4cf147f6fb4d2cdadf3278ab390851`](https://github.com/HSOS6Miyabi/rustdesk-server/commit/2d0896ecea4cf147f6fb4d2cdadf3278ab390851) |
| 已合并的拉取请求 | [#6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) | [#4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4) |
| Docker 镜像 | `ghcr.io/hsos6miyabi/rustdesk-api:v3.0.0-api.103` | `ghcr.io/hsos6miyabi/rustdesk-server:v1.1.17-api.21` |
| 正式 Release | [v3.0.0-api.103](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.103) | [v1.1.17-api.21](https://github.com/HSOS6Miyabi/rustdesk-server/releases/tag/v1.1.17-api.21) |

API 数据库迁移版本为 **282**。两个仓库已经合并并发布 GitHub Release，API 提供 Linux AMD64、Linux ARM64 和 Windows AMD64 发行文件；Server 提供 Linux AMD64 和 Linux ARM64 发行文件。两种 Docker 镜像均已发布 AMD64 和 ARM64 版本，并同时提供版本标签与 `latest`。

## 二、最近已确认成功的自动化检查

| 检查内容 | GitHub Actions |
| --- | --- |
| API Go、Chromium 管理页面、WebClient、WebRTC | [#42689275371](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42689275371) |
| 官方六种桌面安装文件 SHA-256、配置脚本 | [#42688060201](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42688060201) |
| 官方 Linux 1.5.1 客户端 `--deploy`、`--assign`，与实际 API 通信 | [#42688060407](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42688060407) |
| SQLite、MySQL 8.4、PostgreSQL 16 模拟升级及备份恢复 | [#42688060078](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42688060078) |
| Windows、Linux、macOS 官方客户端启动；Android 模拟器安装启动 | [#42688060138](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42688060138) |
| Server Rust 测试、真实 hbbs/hbbr、严格登记、JWT、访问许可与控制角色组合 | [#42690542191](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42690542191) |
| AMD64/ARM64 普通进程部署、组合容器、重启与数据保持 | [#42689263400](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42689263400) |

正式发行工作流：[API #42689275371](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42689275371)、[Server #42690556784](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42690556784)，均已通过程序编译、发行附件上传和双架构 Docker 发布检查。真实公网、多平台账户登录和两台设备间的实际远程控制仍需专门测试。

## 三、功能交付范围

统一台账登记 **F01–F20 共 20 个功能条目**。当前代码已包含账户、群组、联系人、地址簿、设备登记与管理、独立 API 令牌、管理员范围权限、客户端安全策略、控制角色、严格连接许可、多中继与 GeoLite2、WebClient、审计和 CSV、邮件事件通知、客户端安装文件制作及 Docker/普通进程部署。

客户端构建器支持 Windows、Linux、macOS 的 x64 与 ARM64 六种 RustDesk 1.5.1 官方文件。Windows 保留 EXE 文件字节并按官方方式生成带服务器信息的文件名；Linux/macOS 使用保留原始 DEB/DMG 的 ZIP 包，附配置脚本。Android 已在模拟器启动官方 APK。实际客户端登录、远程控制、正式服务器连接和真实系统安装细节仍需测试。

## 四、22 项工作状态及接手顺序

| 编号 | 优先级 | 当前结果与交接任务 |
| --- | --- | --- |
| T001 | P0 | API 与 hbbs 的登记、撤销、身份冲突及撤销后的新连接拒绝检查通过；官方 Linux 客户端部署与分配通过；双终端远程重连待检查 |
| T002 | P0 | 真实 API 进程的密码登录、错误密码、登出与 JWT 再次签发已通过；TOTP、邮箱和恢复码接口测试通过；各操作系统客户端登录及外部身份服务待实测 |
| T003 | P0 | 14 项控制角色信令检查通过；用两台真实设备验证各权限的允许、禁止及默认行为 |
| T004 | P0 | AMD64/ARM64 的登录授权、JWT、访问规则、控制角色、WebRTC SDP/ICE、撤销后新连接拒绝及中继信令测试通过；实际远程操作和公网会话待检查 |
| T005 | P0 | MMDB、Country CSV、节点失效和 reload-geo 测试通过；使用实际 GeoLite2 文件与两个地区的公网中继完成实测 |
| T006 | P0 | 三种临时数据库迁移与恢复检查通过；从正式业务环境获取受保护备份并在独立环境恢复 |
| T007 | P0 | 模拟 SMTP 及隐式 TLS、STARTTLS、身份认证测试通过；完成实际邮箱的邀请、找回密码、验证码与通知收件 |
| T008 | P0 | AMD64/ARM64 普通进程、Docker 与重启测试通过；在目标主机检查域名、证书、权限、数据目录及恢复 |
| T009 | P0 | Chromium 桌面、平板、手机交互和截图已检查，手机端顶部导航单行显示测试通过；目标环境管理账户操作仍待检查 |
| T010 | P0 | 官方 1.4.5/1.5.0/1.5.1 源代码字段检查、1.5.1 多系统启动和 Linux 客户端与实际 API 的设备命令通过；继续验证登录、双重认证及多版本远程连接 |
| T011 | P0 | 跨架构编译与部署检查通过，API 和 Server 的正式发行文件及双架构 Docker 镜像已发布 |
| T012 | P0 | 维护者已批准现有测试条件下发布，两个 PR 均已合并，版本标签、Release 和 Docker 镜像已完成；剩余真实环境检查持续记录 |
| T013 | P1 | 六种官方安装文件及任务管理通过测试；检查每种目标系统的实际安装、配置、签名和远程连接 |
| T014 | P1 | 安全策略心跳、更新、清除、部署许可及设备所有者一致性检查通过；真实客户端的策略效果待检查 |
| T015 | P1 | 独立 API 令牌权限、有效期及撤销已实现，自动测试通过 |
| T016 | P1 | 管理员按用户及用户组范围操作已实现；设备转移同步调整归属并撤销原设备会话，自动测试通过 |
| T017 | P1 | 联系人、共享地址簿、部署许可和回收站批量处理已实现，自动测试通过 |
| T018 | P1 | GeoLite2 Country MMDB 与 CSV 已实现，自动测试通过；正式公网行为计入 T005 |
| T019 | P1 | 登录异常、设备部署、会话结束及文件传输通知已实现；真实收件计入 T007 |
| T020 | P1 | 设备、地址簿与审计筛选、CSV 导出、连接备注已实现；真实团队使用检查待执行 |
| T021 | P2 | 已整理官方 1.5.1 新功能候选；按新版本持续检查 |
| T022 | P2 | 中文页面与移动浏览器交互通过，手机端顶部导航已检查；其他语言及无障碍操作持续完善 |

已发布版本的后续实际环境检查可以先从 **T001、T004、T002、T003** 开始，再检查 **T007、T005、T006、T008、T009、T010、T013、T014、T020**。各项执行步骤、所需资源和完成标准见[T001–T022 逐项测试记录](TASK_VERIFICATION_RECORD.md)。

## 五、仍需提供的实际环境

- 至少两台能互相远程连接的设备，及 Windows/Linux/macOS、Android 和可用的 iOS 客户端；测试账户需与正式业务账户分开。
- 测试环境的完整 API、hbbs、hbbr 访问地址与管理权限；严格模式需要一致的内部密钥和 JWT 设置。
- 至少两个不同地区的公网中继节点，以及有使用权限的 GeoLite2 Country MMDB 或 CSV。
- 可收件的测试邮箱及 SMTP 服务商，独立的 OIDC/LDAP 测试账号（如需测试这些登录方式）。
- 正式业务数据库的受保护备份和独立恢复环境；同时保存 Server 身份密钥、API 配置及客户端构建文件。

没有上述外部环境时，可继续执行单元、浏览器、临时数据库及本地通信测试；这些结果不得填写为真实生产环境测试通过。严禁把用户密码、设备密钥、数据库密码、邮箱密钥或原始令牌写入提交记录和截图。

## 六、接手人的操作步骤

1. 分别检出 API 的 `master` 和 Server 的 `forapi`；Server 额外运行 `git submodule update --init --recursive`。安装正式版本时使用对应的发行标签。
2. 核对两个已发行版本、[API PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6)、[Server PR #4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4) 及相关 GitHub Actions 结果。
3. API 执行 `go test ./...`、`node --test resources/admin/app.test.cjs`，需要浏览器环境时执行 `node --test resources/admin/browser.e2e.cjs`；同时执行 `python3 scripts/verify_task_ledger.py`。
4. Server 执行 `cargo test --lib --tests`、`cargo build --bins`，并运行 `scripts/test_runtime.py`、`scripts/test_strict_combination.py` 和其他对应真实 hbbs 测试；跨架构组合检查使用 `.github/workflows/milestone-preflight.yml`。
5. 对真实环境项目逐项写入测试日期、设备和客户端版本、所用配置、预期与实际行为、CI 或截图地址。更新 [逐项测试记录](TASK_VERIFICATION_RECORD.md) 及 [实施台账](IMPLEMENTATION_LEDGER.md)。
6. 将实际环境检查结果补充到台账。以后发布新版本仍需维护者明确批准，并核对 GitHub Release、Docker 镜像以及 Server 引用的 API 版本。

相关文档：[架构说明](ARCHITECTURE.md)、[部署说明](DEPLOYMENT.md)、[数据库备份恢复](DATABASE_BACKUP_RESTORE.md)、[客户端安装文件](CLIENT_BUILDER.md)、[API 接口](API.md)、[测试说明](TESTING.md)、[官方功能研究](UPSTREAM_FEATURE_WATCH.md)。
