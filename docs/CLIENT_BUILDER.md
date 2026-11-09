# Windows 客户端安装程序制作与管理

客户端安装程序构建器允许管理员保存自己的 RustDesk 服务器配置，为 Windows 用户准备可以直接取得服务器设置的安装程序，并查看生成历史及下载状态。当前支持 **Windows x64 和 Windows ARM64 的官方 RustDesk 1.5.1 EXE**。功能已通过 Go、Chromium 与官方文件 SHA-256 检查，正式安装前仍应使用实际 Windows 设备验证。

## 使用前的准备

需要准备已经运行的 RustDesk API、`hbbs`、`hbbr`，以及服务器公钥：

- **ID 服务器地址**：例如 `desk.example.com:21116`。请使用客户端能够访问的地址。
- **服务器公钥**：`hbbs` 数据目录中的 `id_ed25519.pub` 内容。公钥解码后应为 32 字节。
- **API 地址**：例如 `https://desk.example.com`。建议通过 HTTPS 对外提供服务。
- **中继服务器地址**：例如 `relay.example.com:21117`，也可以留空，由 ID 服务器提供中继地址。

配置 ID 服务器和中继地址前，应先确认 DNS、TLS 证书和对应端口能够在用户所在的网络访问。客户端构建页面要求有效的管理员登录会话。

## 在管理页面制作安装程序

1. 登录 API 管理控制台，打开“客户端安装包构建”。
2. 输入配置名称、选择 Windows x64 或 Windows ARM64，再填写 ID 服务器、公钥、API 和中继地址。
3. 点击“保存配置”。已保存的配置会显示在下方，可再次编辑，或者创建新的配置。
4. 对目标配置点击“创建安装包任务”。任务列表显示等待处理、正在处理、安装包已准备好或处理失败。
5. 任务准备完成后点击“下载安装程序”。失败任务可以重新处理。

系统会下载对应平台的官方 RustDesk 1.5.1 EXE，核对文件大小及 SHA-256，并将服务器配置编码在官方客户端可识别的文件名中。**原始 EXE 内容不会被修改。** 服务器在下载文件给管理员之前还会重新核对其 SHA-256，以检查已保存的文件。

已创建的任务保留当时的配置副本。后来编辑或删除配置，不会更改此前生成的安装程序。

## 文件保存及部署方式

服务器默认将生成的文件保存在 `data/client-builds/<任务编号>/`，可以通过环境变量 `RUSTDESK_CLIENT_BUILD_DIR` 指定其他目录。

- **普通服务器进程**：为运行 API 的系统账户准备可写且持久保存的目录，并在备份中包含该目录。
- **Docker**：默认目录位于容器内的 `/app/data/client-builds`。挂载并保留 `/app/data`，更新镜像或重新创建容器后即可继续访问已经准备好的安装程序。也可以将自定义目录挂载到容器内。
- **访问权限**：构建目录使用受限制的文件权限。下载接口需管理员登录令牌，完成任务只允许从下载接口读取。
- **数据备份**：除安装文件外，请一并备份数据库中的 `client_build_profiles` 和 `client_build_tasks`。数据库保存任务状态和下载文件的校验值。

构建过程中的文件来源固定为官方版本的 GitHub Release，任务不能提交任意下载地址。文件验证不成功时，任务状态会记录原因，不会提供该文件供下载。

## 管理 API

需要有效的 API 管理后台登录令牌，使用请求头 `api-token`。接口路径如下：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/api/admin/client_builder/profiles?page=1&page_size=20` | 查看已保存的配置 |
| POST | `/api/admin/client_builder/profiles` | 新建或修改配置 |
| POST | `/api/admin/client_builder/profiles/delete` | 删除指定配置 |
| GET | `/api/admin/client_builder/tasks?page=1&page_size=20` | 查看任务历史 |
| POST | `/api/admin/client_builder/tasks` | 创建构建任务 |
| POST | `/api/admin/client_builder/tasks/retry` | 重新处理失败任务 |
| GET | `/api/admin/client_builder/tasks/:id/download` | 下载已经准备好的安装程序 |

保存配置示例：

```json
{
  "name": "技术支持团队",
  "platform": "windows-x64",
  "version": "1.5.1",
  "host": "desk.example.com:21116",
  "public_key": "替换为自己的服务器公钥",
  "api_server": "https://desk.example.com",
  "relay_server": "relay.example.com:21117"
}
```

创建任务示例为 `{"profile_id": 12}`，重新处理任务示例为 `{"id": 8}`。修改配置时需携带该配置的 `id`。

任务字段 `status` 可能为 `pending`（等待）、`building`（正在处理）、`ready`（可下载）或 `failed`（失败）。任务记录包含平台、版本、文件名、SHA-256、文件大小以及处理时间。

## 检查与限制

源码检查和自动测试位于 `service/client_build_test.go`、`resources/admin/app.test.cjs` 和 `resources/admin/browser.e2e.cjs`。对应 GitHub Actions 工作流是 `.github/workflows/verify-client-build.yml`，同时实际下载官方 Windows x64、ARM64 安装程序，检查文件头、大小和 SHA-256。

当前预设版本固定为 RustDesk 1.5.1。只有在更新官方文件的地址、校验值和对应测试后，才能允许选择新的客户端版本。macOS、Linux 的专用安装包制作与真实跨平台安装仍需进一步完成。

在 Windows 上交付时，应检查安装程序的数字签名、实际安装结果、服务器设置、登录和远程连接。未经测试的自定义安装程序不得直接替换已经在使用的正式客户端。

正式里程碑发布仍须由维护者明确确认；构建器生成的个人安装程序不属于本项目的 GitHub Release。
