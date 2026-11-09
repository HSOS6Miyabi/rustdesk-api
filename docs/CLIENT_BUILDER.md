# RustDesk 客户端安装文件制作与管理

管理员可以在网页中保存 RustDesk 服务器配置，制作带有配置说明的官方客户端安装文件，查看处理记录并下载文件。当前固定使用 **RustDesk 1.5.1**，支持 Windows、Linux、macOS 的 x64 与 ARM64 架构。系统对所有官方安装文件核对大小与 SHA-256，并对生成的文件再次检查完整性。

本功能已通过安装文件下载、配置编码、ZIP 文件内容、脚本语法、Linux 模拟安装命令及浏览器操作测试。实际客户端安装、系统权限及远程桌面连接仍需要相应系统上的真实设备验证。

## 各操作系统的安装文件

| 系统与架构 | 下载文件 | 服务器配置方式 |
| --- | --- | --- |
| Windows x64、Windows ARM64 | 官方 `.exe` | 安装文件名使用 RustDesk 官方客户端支持的服务器配置格式 |
| Linux x64、Linux ARM64 | 包含官方 `.deb`、`install-and-configure.sh` 和说明的 `.zip` | 在 Debian/Ubuntu 中安装原始 DEB 后，通过 RustDesk 命令行保存服务器配置 |
| macOS x64、macOS ARM64 | 包含官方 `.dmg`、`configure-rustdesk.command` 和说明的 `.zip` | 将 RustDesk.app 安装至“应用程序”目录后，通过 RustDesk 命令行保存服务器配置 |

Windows 文件保留官方 EXE 原始字节。Linux 与 macOS 的 ZIP 文件中也保留原始 DEB 或 DMG 文件，配置脚本作为单独的文本文件提供。ZIP 的 SHA-256 用于核对整个压缩包，压缩包内的 `README.txt` 记录官方安装文件的 SHA-256。

Linux 的自动安装脚本适用于 Debian 和 Ubuntu；其他 Linux 发行版应使用 RustDesk 官方对应格式的安装程序并手动设置服务器。

## 准备服务器配置

在创建配置前，需要已经可以访问的 RustDesk API、`hbbs` 和 `hbbr`，以及以下资料：

- **ID 服务器**，例如 `desk.example.com:21116`，填写客户端可以连接的地址。
- **服务器公钥**，从 `hbbs` 数据目录的 `id_ed25519.pub` 读取，Base64 解码结果应为 32 字节。
- **API 地址**，例如 `https://desk.example.com`。对外访问建议使用 HTTPS。
- **中继服务器**，例如 `relay.example.com:21117`。若由 ID 服务器自动提供中继地址，可以留空。

设置之前应检查域名、证书及相关端口的访问情况。创建和下载任务需要有效的系统管理员登录会话。

## 网页使用方法

1. 登录管理控制台，打开“客户端安装包构建”。
2. 输入配置名称，选择目标操作系统及架构，填写服务器地址、公钥、API 地址和可选的中继地址，点击“保存配置”。
3. 在“已保存的配置”中选择“创建安装包任务”，然后查看下方的任务状态。
4. 任务状态为“安装包已准备好”时，点击“下载安装程序”。若任务失败，查看显示的原因后使用“重新处理”。
5. 在目标计算机上按以下方法安装和配置，最后检查服务器地址、登录和远程连接。

**Windows**：直接使用下载的 EXE 安装，检查文件数字签名。安装后的服务器配置由 RustDesk 官方客户端识别安装程序文件名中的内容。

**Linux**：解压 ZIP，在 Debian 或 Ubuntu 的终端进入解压目录，执行：

```bash
sh ./install-and-configure.sh
```

脚本会通过系统的 `apt-get` 安装同目录官方 DEB，再调用 RustDesk 的 `--option` 命令保存服务器参数并逐项读取核对。安装与配置需要 `sudo` 权限，客户端服务需要正常运行；如果设置没有生效，脚本会停止并显示检查结果。

**macOS**：解压 ZIP，打开 DMG 并将 RustDesk.app 放入“应用程序”目录，然后在终端进入解压目录执行：

```sh
sh ./configure-rustdesk.command
```

配置脚本通过已安装的 RustDesk 命令行保存服务器参数，再次读取参数核对结果，需要系统管理员权限和可访问的 RustDesk 服务。设置未成功保存时，脚本会显示错误。DMG 与应用程序文件本身保持原样。首次运行仍需按 macOS 提示授予远程控制所需的系统权限。

## 任务记录、保存及备份

创建任务时，服务器会将当时的配置另存至该任务。后来修改或删除配置，已经创建的任务不会改变。任务状态包括 `pending`、`building`、`ready` 和 `failed`。如果处理进程中断，服务器会标记长时间未完成的任务，管理员可以重新提交。

文件默认保存在 `data/client-builds/<任务编号>/`，也可以通过环境变量 `RUSTDESK_CLIENT_BUILD_DIR` 自行指定。

- 普通服务器进程运行时，应确保 API 服务账户有权写入该目录，并将其加入备份。
- Docker 部署默认使用容器内 `/app/data/client-builds`。持久挂载 `/app/data`，在重新创建容器后保留安装文件。
- 同时备份数据库中的 `client_build_profiles`、`client_build_tasks` 和安装文件目录。数据库保存处理状态、最终文件大小与 SHA-256。
- 文件只允许从经过管理员身份验证的接口下载。服务器下载官方文件与向管理员提供文件时均验证 SHA-256，并直接发送已通过核对的文件内容。

构建任务只允许使用固定的官方版本和下载地址，不接受任意第三方安装文件链接。当前版本更新时应同步修改官方文件校验值和自动测试。

## 管理 API

发送管理后台登录令牌，使用 `api-token` 请求头。接口如下：

| 请求方法 | 地址 | 用途 |
| --- | --- | --- |
| GET | `/api/admin/client_builder/profiles?page=1&page_size=20` | 查看配置 |
| POST | `/api/admin/client_builder/profiles` | 新建或修改配置 |
| POST | `/api/admin/client_builder/profiles/delete` | 删除配置 |
| GET | `/api/admin/client_builder/tasks?page=1&page_size=20` | 查看任务 |
| POST | `/api/admin/client_builder/tasks` | 创建任务 |
| POST | `/api/admin/client_builder/tasks/retry` | 重新提交失败任务 |
| GET | `/api/admin/client_builder/tasks/:id/download` | 下载准备好的安装文件 |

新建配置示例：

```json
{
  "name": "技术支持团队",
  "platform": "linux-x64",
  "version": "1.5.1",
  "host": "desk.example.com:21116",
  "public_key": "替换为自己的服务器公钥",
  "api_server": "https://desk.example.com",
  "relay_server": "relay.example.com:21117"
}
```

`platform` 支持 `windows-x64`、`windows-arm64`、`linux-x64`、`linux-arm64`、`macos-x64`、`macos-arm64`。新建任务使用 `{"profile_id":12}`，重新提交失败任务使用 `{"id":8}`。修改已保存配置时需要提供该配置的 `id`。

## 检查方式及发布条件

相关代码位于 `service/client_build.go`、`service/client_build_worker.go`、`service/client_packages.go`。单元和模拟安装检查位于 `service/client_build_test.go` 与 `service/client_packages_test.go`；网页测试位于 `resources/admin/app.test.cjs` 和 `resources/admin/browser.e2e.cjs`。GitHub Actions 的 `.github/workflows/verify-client-build.yml` 会实际下载六个平台的官方文件并核对 SHA-256。

在真实系统交付前，仍需分别检查 Windows 安装程序的数字签名、Linux 发行版依赖、macOS 应用程序权限、登录流程和远程桌面连接。只有全部发布前检查完成并获得明确批准，才可以创建本项目的正式 Release。
