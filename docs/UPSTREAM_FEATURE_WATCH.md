# RustDesk 客户端版本与功能研究

更新日期：2026 年 11 月 9 日。

## 当前核查版本

RustDesk 官方稳定版的最新发布标签为 [1.5.1](https://github.com/rustdesk/rustdesk/releases/tag/1.5.1)，发布日期是 2026 年 11 月 1 日。本项目保持对官方 1.5.x 客户端的兼容目标。客户端新功能需要结合客户端的实现条件确定是否需要 API 或 hbbs 修改。

参考材料：

- [RustDesk 1.5.1 发布说明](https://github.com/rustdesk/rustdesk/releases/tag/1.5.1)
- [RustDesk 官方高级设置](https://rustdesk.com/docs/zh-cn/self-host/client-configuration/advanced-settings/)
- [RustDesk 客户端开源仓库](https://github.com/rustdesk/rustdesk)
- [RustDesk Server 开源仓库](https://github.com/rustdesk/rustdesk-server)

## 后续功能候选

| 优先程度 | 客户端版本及功能 | 使用场景 | 研究与开发位置 | 下一项可执行检查 |
| --- | --- | --- | --- | --- |
| 高 | 1.5.1，受信任设备 V2，兼容旧信任记录，[PR #16548](https://github.com/rustdesk/rustdesk/pull/16548) | 团队成员反复登录时按设备管理可信凭据 | API 的 `service/totp.go`、`service/email_login.go`、`service/device_policy.go`，以及客户端登录协议 | 使用升级前后的 Windows 和 Linux 客户端检查可信设备续期、撤销与丢失设备时的处理 |
| 高 | 1.5.1，更新元数据签名验证，[PR #15914](https://github.com/rustdesk/rustdesk/pull/15914) | 客户端下载更新时验证来源及完整性 | `service/client_build_worker.go`、客户端构建流程、升级文档 | 保留官方安装包原始内容，检查 SHA-256，并在 Windows 真实安装环境中验证签名与更新 |
| 中 | 1.5.1，地址簿页面打开 Web 控制台入口，[PR #16680](https://github.com/rustdesk/rustdesk/pull/16680) | 从桌面客户端快速进入设备与地址簿管理页面 | `resources/admin/app.js`、`http/router/admin.go`、`resources/web2` | 检查 Web 控制台登录地址、不同账户权限和浏览器跳转 |
| 中 | 1.5.1，Android 通过 SAF 目录直接传输文件，[PR #16644](https://github.com/rustdesk/rustdesk/pull/16644) | Android 设备直接选择系统授权的目录处理文件传输 | `http/controller/api/audit.go`、`service/audit_ownership.go` | 在 Android 端确认文件传输记录字段、权限限制及目标路径显示 |
| 中 | 1.5.1，快捷键增强，[PR #14933](https://github.com/rustdesk/rustdesk/pull/14933) | 远程控制时使用键盘快捷键提高效率 | 现有的控制角色、连接权限及浏览器客户端 | 检查键盘控制权限关闭时快捷键是否受限制 |
| 中 | 1.5.0 起，`enable-tcp-punch` 与 `relay-fallback-delay` | 内网穿透、WebRTC 和中继之间的连接体验调整 | `model/device_policy.go`、`service/device_policy.go`、Server 的 `src/relay_policy.rs` | 在两种公网网络环境中测量连接成功率及回退表现，随后选择是否加入策略页面 |
| 中 | 1.4.1 起，`enable-udp-punch`、`enable-ipv6-punch` | 支持 IPv6 与 UDP 网络的设备远程连接 | `model/device_policy.go`、`service/device_policy.go`、`src/rendezvous_server.rs` | 验证客户端平台和服务器支持情况，再增加可撤销的策略配置 |
| 低 | 1.5.1，更多 macOS/Wayland 兼容修复 | Linux 和 macOS 远程桌面在不同显示环境中稳定运行 | 客户端兼容性测试及 `docs/TESTING.md` | 使用可用的 macOS、X11、Wayland 机器执行视频、鼠标、键盘及文件传输测试 |

## 本项目现有功能与候选项目的关系

本项目已经实现的设备策略、控制角色、集中连接许可、活动会话、审计日志和地址簿共享分别具有独立接口。新增候选功能优先使用已有协议和模型，避免影响用户现有的设备连接设置。

当前策略管理已经支持 `access-mode`、`enable-keyboard`、`enable-clipboard`、`enable-file-transfer` 和 `approve-mode`。官方文档说明客户端设置会受到覆盖设置、策略、用户设置和默认设置四级优先顺序影响；新增功能应当先明确其行为，再进行多平台验证。

可视化客户端构建目前采用官方 1.5.1 Windows 安装程序，保留原程序文件内容，在客户端可识别的文件名中编码服务器信息。跨平台原生打包、真正的安装验证及不同平台签名检查仍需单独完成。

## 持续检查方法

每次官方发布稳定版本后，执行以下工作：

1. 阅读新的发布说明和相关源代码修改，更新本文记录的最新版本、发布日期和功能候选。
2. 对照 `docs/FEATURE_MATRIX.md` 与 `docs/IMPLEMENTATION_LEDGER.md`，将已经实现的功能和新候选功能分别登记。
3. 对涉及客户端配置、身份认证和连接协议的修改增加自动测试，再在受支持的客户端系统上检查真实行为。
4. 仅在两个仓库完成当前里程碑要求且获得明确发布确认后，才更新正式发行版本。

即使没有新的稳定版本，也应当在每个月检查一次官方高级设置文档与相关已合并功能，确认有无需要新增的兼容性检查。


## 官方 Server 主分支兼容性检查

2026 年 11 月 9 日核对了官方 Server 主分支提交 [680b6039](https://github.com/rustdesk/rustdesk-server/commit/680b60395ecf99a76f74a1d934e453bb6e5e0aa2)。该修改完善 IPv4 与 IPv4 映射地址在中继拒绝列表中的匹配。本项目 Server 的 `src/relay_server.rs` 已包含相应处理与测试。后续更新官方稳定版本时，继续检查此部分与连接安全设置的兼容情况。
