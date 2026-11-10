# 消息推送与 IP 地理位置

RustDesk API 支持每个用户自行设置消息推送渠道、订阅设备事件，并在管理页面中查看 IP 地址对应的地理位置。所有渠道使用统一的消息内容格式，重要信息会优先出现在标题和正文开头。

## 1. 支持的推送方式

| 推送方式 | 需要填写的设置 | 说明 |
| --- | --- | --- |
| MessagePusher | 服务地址、发送密钥 | 调用 MessagePusher 的 `POST /v1/send`。由 MessagePusher 将消息转发到用户在其平台中配置的 Telegram、邮件等渠道。 |
| SMTP 邮件 | SMTP 主机、端口、TLS/STARTTLS（可信内网还可选择不加密）、用户名、密码、发件邮箱、收件邮箱 | 用户可以使用自己的邮箱服务器。 |
| Server酱 Turbo | SendKey，必要时填写自建 API 地址 | 调用 `POST /{SendKey}.send`。默认使用官方的 `https://sctapi.ftqq.com`。 |

登录控制台后打开 **账户 → 消息推送**，添加渠道并填写设置。一个用户可以添加多个渠道，例如同时向个人 Telegram 和邮箱发送设备离线通知。

MessagePusher 的发送密钥和 SMTP 密码经过 AES-GCM 加密后保存。设置页面仅显示凭据是否已配置。重新编辑渠道时，密码留空即可沿用原来的值。

## 2. 订阅事件

每个渠道都可以独立选择以下事件：

| 事件代码 | 通知内容 |
| --- | --- |
| `device.registered` | 新设备登记 |
| `device.online` | 设备上线 |
| `device.offline` | 设备离线 |
| `connection.started` | 远程连接建立 |
| `connection.ended` | 远程连接结束 |
| `audit.file` | 文件传输 |
| `audit.alarm` | 设备安全告警 |
| `deployment.changed` | 设备部署状态变更 |
| `session.disconnect_requested` | 管理员请求结束远程连接 |
| `session.ended` | 已按管理请求结束远程会话 |
| `login.failure` | 账户登录失败 |

在线状态以已通过设备身份验证的心跳为依据。首次确认在线以及离线后再次在线时，会产生上线事件；连续 120 秒未收到有效心跳，系统会将设备标记为离线。后台每 30 秒检查一次离线设备。

远程连接、文件传输和安全告警依赖 RustDesk 客户端报告对应审计事件。客户端未报告的事件，系统无法补充其连接细节。

## 3. 消息内容与预览

通知标题优先包含事件、设备名称、设备 ID、连接方设备，以及已知来源 IP 和地理位置。正文接着列出有关用户、发生时间、设备系统信息、连接编号、认证方式和事件说明。尚未采集到的字段会自动略过。

例如：

```text
标题：[RustDesk · 远程连接建立] 办公室电脑 (123456789) ← 客服电脑 (987654321)

事件：远程连接建立
设备：办公室电脑 (123456789)
连接方设备：客服电脑 (987654321)
来源 IP：192.168.1.15（局域网地址）
设备所有者：alice (ID 3)
连接方账户：support (ID 5)
发生时间：2026-11-10 11:30:00 +08:00
设备主机名：office-pc
设备登录用户：alice
操作系统：Windows 11
连接编号：4031
```

某些事件只包含设备 ID 或来源 IP，系统会显示现有的信息，不会自行补写用户名或地理位置。

## 4. 启用 GeoIP 查询

RustDesk API 使用本地 MaxMind 格式的城市数据库。请从 MaxMind 获取 GeoLite2-City 数据文件 `GeoLite2-City.mmdb`，并定期更新该文件。

使用普通服务器部署时，可以按以下方式设置环境变量：

```bash
export RUSTDESK_API_PUSH_ENCRYPTION_KEY='填写至少16个字符的随机密钥'
export RUSTDESK_API_GEOIP_DB='/var/lib/rustdesk-api/GeoLite2-City.mmdb'
./apimain
```

Docker 部署时，将数据库文件挂载到容器的只读目录，并在容器的环境变量中设置：

```yaml
services:
  rustdesk-api:
    environment:
      RUSTDESK_API_PUSH_ENCRYPTION_KEY: "填写至少16个字符的随机密钥"
      RUSTDESK_API_GEOIP_DB: "/app/data/GeoLite2-City.mmdb"
    volumes:
      - "./data:/app/data"
```

上面的 Compose 片段用于说明新增的环境变量和文件位置。实际部署时，请保留原来的服务配置。

配置完成后重新启动 RustDesk API。以下位置会自动显示带括号的地理位置：

- 设备管理、登录记录、我的登录记录。
- 连接记录、文件传输、安全告警和管理操作记录。
- 管理页面中显示的 IP 地址和 IP 白名单条目，以及使用 IP 地址表示的服务器地址。
- MessagePusher、SMTP 与 Server酱通知中的 IP 地址。

例如：`8.8.8.8（美国 · 加利福尼亚州）`。实际结果以本地数据库为准，数据库缺少对应记录时显示 `（位置未知）`。局域网地址和回环地址分别显示 `（局域网地址）`、`（本地回环地址）`。CIDR 网段代表一个地址范围，界面会标记为 `（IP 地址段）`。

GeoIP 查询只读取本地文件，无需调用第三方位置查询服务。原有数据库中的 IP 地址保持不变，地理位置仅在显示或生成通知时补充。

## 5. 异步发送与失败重试

事件处理接口将消息交给内存缓冲区，然后继续处理原来的业务请求。后台程序将需要发送的消息写入数据库，按照渠道处理发送任务。

系统使用两个发送工作协程，收到新任务时可立即开始处理；平时每隔 3 秒检查待发送任务。每条消息最多尝试发送 5 次，失败时依次延迟 10、20、40 和 80 秒重试。达到上限后，用户可以在发送记录中手动重试。

发送状态包括等待发送、正在发送、等待重试、暂停发送、发送成功及发送失败。关闭渠道时会暂停待处理的消息；重新启用后可以继续发送。

MessagePusher 返回 HTTP 202 代表它已接收发送任务；用户需要在 MessagePusher 中查看最终外部渠道的发送结果。

内存缓冲区容量为 4096 个事件。进程异常终止、缓冲区已满或数据库无法写入时，尚未保存的事件可能丢失。系统不会因此延迟设备心跳或连接审计请求。

## 6. 安全设置

`RUSTDESK_API_PUSH_ENCRYPTION_KEY` 用于加密每位用户的推送密钥和 SMTP 密码。也可沿用已经设置的 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`，但建议为推送服务配置独立密钥。必须妥善保存密钥，更换密钥前应先处理原有加密凭据。

默认只接受 HTTPS 推送地址，且拒绝连接到局域网、回环地址和其他受限 IP 地址，以减少用户配置任意地址带来的风险。

如果 MessagePusher 或 SMTP 服务部署在可信的局域网内，并且确实需要访问内部地址，可以由服务器管理员添加：

```bash
export RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS=1
```

启用后，请同时限制服务器可访问的网络和推送服务的账户权限。公开访问的服务仍应使用 HTTPS 或 TLS。可信的内部 SMTP 服务可以在启用上述环境变量后，选择不加密连接。

## 7. 管理 API

以下接口都位于 `/api/admin` 下，需要在请求头中填写当前用户的 `api-token`。服务端通过登录用户身份限制数据访问范围。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/push/events` | 获取可以订阅的事件 |
| GET | `/push/channels` | 获取本人渠道 |
| POST | `/push/channels` | 创建或更新本人渠道 |
| POST | `/push/channels/delete` | 删除渠道 |
| POST | `/push/channels/test` | 提交测试通知 |
| GET | `/push/history` | 查询本人发送记录 |
| POST | `/push/history/retry` | 重新发送失败的消息 |
| POST | `/geoip/lookup` | 批量查询 IP 地理位置，最多 256 个地址 |

创建 MessagePusher 渠道的示例：

```json
{
  "kind": "messagepusher",
  "name": "个人消息通知",
  "enabled": true,
  "base_url": "https://push.example.com",
  "api_key": "mpk_YOUR_SEND_KEY",
  "events": ["device.online", "device.offline", "connection.started"]
}
```

批量查询 GeoIP 的示例请求：

```json
{"ips": ["8.8.8.8", "192.168.1.15", "127.0.0.1"]}
```

返回的 `locations` 对象以原始 IP 地址为键，值为地理位置或地址类型。
