# 消息推送与 IP 地理位置

RustDesk API 支持每个用户自行设置消息推送渠道、订阅设备事件，并在管理页面中查看 IP 地址对应的地理位置。所有渠道使用统一的消息内容格式，重要信息会优先出现在标题和正文开头。

## 1. 支持的推送方式

| 推送方式 | 需要填写的设置 | 说明 |
| --- | --- | --- |
| MessagePusher | 服务地址、发送密钥 | 调用 MessagePusher 的 `POST /v1/send`。由 MessagePusher 将消息转发到用户在其平台中配置的 Telegram、邮件等渠道。 |
| SMTP 邮件 | SMTP 主机、端口、TLS/STARTTLS（可信内网还可选择不加密）、用户名、密码、发件邮箱、收件邮箱 | 用户可以使用自己的邮箱服务器。 |
| Server酱 Turbo | SendKey，必要时填写自建 API 地址 | 调用 `POST /{SendKey}.send`。默认使用官方的 `https://sctapi.ftqq.com`。 |
| 自定义 HTTP JSON | 完整 HTTPS 接收地址；可选 Bearer 密钥 | 直接向指定地址发送 `POST` 请求，请求正文为 JSON，接收方返回 2xx 视为成功。 |

登录控制台后打开 **账户 → 消息推送**，添加渠道并填写设置。一个用户可以添加多个渠道，例如同时向个人 Telegram、邮箱和自己的 HTTP 接口发送设备离线通知。

MessagePusher 的发送密钥、自定义 HTTP 渠道的 Bearer 密钥和 SMTP 密码经过 AES-GCM 加密后保存。设置页面仅显示凭据是否已配置。重新编辑渠道时，密码留空即可沿用原来的值。

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

在线状态以已通过设备身份验证的心跳为依据。首次确认在线以及离线后再次在线时，会产生上线事件；连续 120 秒未收到有效心跳，系统会将设备标记为离线。系统为每个在线设备根据最近一次有效心跳计算离线截止时间，并在最早的截止时间到达时检查到期设备。收到新心跳后会立即更新到期时间。

远程连接、文件传输和安全告警依赖 RustDesk 客户端报告对应审计事件。客户端未报告的事件，系统无法补充其连接细节。

## 设备在线状态显示

用户控制台、管理员控制台及 WebClient v2 的设备列表，都在设备名称或设备 ID 前显示状态圆点。

- **绿色**：最近 120 秒内收到了已验证的设备心跳，当前记录为在线。
- **红色**：设备离线或尚未收到有效心跳。
- **灰色**：网页暂时无法获取状态。

设备管理、共享设备、地址簿、最近出现的设备、远程连接记录、文件传输记录、安全告警，以及包含设备 ID 的管理页面均使用相同的状态数据。页面保留现有列表字段，直接在设备信息前显示圆点。

网页加载设备列表时，将这一页的设备 ID 与待解析 IP 地址一起提交给 `POST /api/admin/geoip/lookup`。服务端按当前登录账户的设备所有权、共享权限和管理范围检查访问权限，然后批量查询在线状态。每次最多处理 256 台设备和 256 个 IP 地址。页面仅在打开或手动刷新时读取状态，不会定时请求服务器。设备本身继续通过 RustDesk 的心跳上报状态。

为减轻大量设备心跳对数据库的压力，服务端在内存中合并同一设备短时间内重复且 IP 未改变的心跳，最多每 15 秒更新一次在线时间。设备 IP 发生变化会立即触发更新。事件处理按设备分配到四个工作协程，同一设备的事件按顺序处理。离线状态通过最早到期时间触发检查。



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

系统使用两个发送工作协程。新消息保存到数据库后，立即发送唤醒信号；工作协程空闲时等待事件。需要延迟重试时，系统查询数据库中最早到期的任务，并设置对应的定时器。每条消息最多尝试发送 5 次，失败时依次延迟 10、20、40 和 80 秒重试。达到上限后，用户可以在发送记录中手动重试。

发送状态包括等待发送、正在发送、等待重试、暂停发送、发送成功及发送失败。关闭渠道时会暂停待处理的消息；重新启用后可以继续发送。

MessagePusher 返回 HTTP 202 代表它已接收发送任务；用户需要在 MessagePusher 中查看最终外部渠道的发送结果。

设备事件使用容量为 4096 的内存缓冲区，原有管理员邮件事件使用容量为 1024 的内存缓冲区。进程异常终止、缓冲区已满或数据库无法写入时，尚未保存的事件可能丢失。系统不会因此延迟设备心跳或连接审计请求。

队列中的未完成任务会保存在数据库中。服务重新启动时，工作协程首先检查待发送任务和上次中断的发送记录；数据库中没有到期任务时，不会定时查询空闲队列。设备离线检查也使用到期定时器。原有管理员邮件通知采用相同的发送调度方式。

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
| POST | `/geoip/lookup` | 批量查询 IP 地理位置和授权设备的在线状态，各最多 256 项 |

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
{"ips": ["8.8.8.8", "192.168.1.15", "127.0.0.1"], "device_ids": ["123456789", "987654321"]}
```

返回的 `locations` 对象以原始 IP 地址为键，值为地理位置或地址类型；`device_status` 对象以设备 ID 为键，`true` 表示在线，`false` 表示离线或无权查询。


## 8. SSE 实时事件接口

登录控制台，打开 **账户 → 消息推送 → SSE 实时订阅**，输入用途名称，设置有效期并生成密钥。密钥只会显示一次，请复制保存。普通用户可以为自己的账户创建密钥，无需管理员权限。

新生成的密钥以 `rdapi_` 开头，拥有 `events.read` 权限，只能接收与密钥所属账户有关的事件。服务端仅保存密钥的 SHA-256 摘要。可以随时在同一页面撤销密钥。

### 连接接口

- **方法：** `GET`
- **地址：** `/api/events/v1/stream`
- **认证：** `Authorization: Bearer <API_KEY>`
- **返回类型：** `text/event-stream`
- **事件筛选：** 可选查询参数 `events`，多个事件代码用英文逗号分隔；省略时订阅所有可用事件。

例如，只订阅远程连接建立、远程连接结束：

```bash
curl -N --fail-with-body \
  -H "Accept: text/event-stream" \
  -H "Authorization: Bearer rdapi_替换为你的密钥" \
  "https://desk.example.com/api/events/v1/stream?events=connection.started,connection.ended"
```

收到事件时，服务器会发送类似内容：

```text
event: connection.started
data: {"id":"a246f8c4...","type":"connection.started","occurred_at":"2026-11-13T02:30:00Z","device_id":"123456789","from_peer":"987654321","connection_id":4031,"title":"[RustDesk · 远程连接建立] 办公室电脑","body":"事件：远程连接建立\n设备：办公室电脑"}
```

`data` 是一行合法 JSON。字段说明：

| 字段 | 说明 |
| --- | --- |
| `id` | 事件标识，同一事件在推送重试时保持一致，客户端可据此去除重复消息 |
| `type` | 事件代码，与 `event:` 行一致 |
| `occurred_at` | 事件发生时间，采用 UTC ISO 8601 格式 |
| `device_id` / `from_peer` | 目标设备 ID、连接方设备 ID；无对应信息时省略 |
| `source_ip` | 来源 IP；无对应信息时省略 |
| `connection_id` | 连接编号；没有连接编号时省略 |
| `detail` | 补充说明；没有说明时省略 |
| `title` / `body` | 与其他通知渠道一致的标题和详细正文 |

连接建立事件使用现有的 `connection.started`。RustDesk 客户端向 `/api/audit/conn` 上报连接建立记录，服务器保存成功后触发通知；无须另行订阅同义事件。

### 连接管理与性能

SSE 请求会保持 HTTP 连接，不需要客户端反复请求事件列表。服务器每 20 秒发送一条以冒号开头的心跳注释，用于维持空闲连接。客户端应在连接断开后重新建立连接，并继续携带相同的 Authorization 请求头。

每个账户最多同时建立 **5 条** SSE 连接，单个服务器进程最多 **2048 条**。每条连接最多缓冲 64 条尚未发送的事件。客户端读取过慢且缓冲区耗尽时，服务器会关闭该连接，让客户端重新连接；请求取消、断线或服务停止后会清理对应的订阅记录。服务端每隔约一分钟重新检查密钥权限，撤销的密钥无法继续建立新连接，原有连接会在重新检查时关闭。

**SSE 传送当前在线收到的实时事件**。服务器重启、客户端离线及网络中断期间的消息不会补发。如果业务要求失败后自动重试并保留发送记录，请使用下文的自定义 HTTP JSON 渠道。

API Key 必须放在 Authorization 请求头中，不要放在 URL 查询参数中。浏览器原生 `EventSource` 不能设置 Authorization 头；可以使用支持流式读取的 `fetch` 接口或带请求头的 SSE 客户端库。反向代理需要允许长时间 HTTP 连接，并关闭该接口的响应缓冲。服务端已经设置 `X-Accel-Buffering: no`。

### 多实例部署

如果多个 API 服务器进程共同提供服务，可以为各进程配置同一个 Redis 服务：

```yaml
redis:
  addr: "redis.example.internal:6379"
  password: "填写 Redis 密码"
  db: 0
```

配置 Redis 后，各 API 进程会通过 Redis Pub/Sub 转发事件，并记录当前进程正在订阅的事件类型。订阅记录设置了过期时间，进程异常退出后也会自动清理。未配置 Redis 时，SSE 只接收当前 API 进程处理的事件。多实例部署建议配置 Redis，以免连接到其他实例的用户错过事件。

## 9. 自定义 HTTP JSON 推送

打开 **账户 → 消息推送**，选择 **HTTP JSON**，填写接收地址、订阅事件，以及接收服务器需要的可选 Bearer 密钥。

也可以使用管理 API 创建渠道：

```http
POST /api/admin/push/channels
api-token: <已登录账户的会话令牌>
Content-Type: application/json

{
  "kind": "webhook",
  "name": "自动化事件接收",
  "enabled": true,
  "base_url": "https://events.example.com/rustdesk",
  "api_key": "可选的接收方密钥",
  "events": ["connection.started", "connection.ended", "device.offline"]
}
```

服务器会向 `base_url` 指定的**完整地址**直接发送 `POST` 请求，不会拼接额外的路径。请求头包含 `Content-Type: application/json`；配置密钥后，另含 `Authorization: Bearer <密钥>`。请求正文使用上文的 JSON 事件字段。手动测试通知的 `type` 为 `test.manual`。

接收服务器只需返回任意 **2xx** 状态码，即可确认已接收事件。发送失败、超时或返回其他状态码时，任务保留在数据库中，最多发送 5 次，之后可在控制台手动重试。同一个事件的 `id` 在重试期间保持不变，接收方应据此避免重复处理。

推送地址默认必须使用 HTTPS。服务器会检查解析后的目标 IP，拒绝发送到回环、局域网以及其他受限地址。若确实需要访问可信内网，可由服务器管理员设置 `RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS=1`，并限制接收服务器的访问权限。服务端不会跟随 HTTP 重定向，也不会将接收方返回的敏感正文写入发送日志。

## 10. SSE 密钥管理 API

以下接口位于 `/api/admin` 下，需要使用用户登录时取得的 `api-token` 请求头：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/push/stream-keys` | 查看本人创建的 SSE 密钥，包括名称、前缀、过期时间和撤销状态 |
| POST | `/push/stream-keys` | 创建个人 SSE 密钥，请求体为 `{"name":"家庭自动化","expires_in_days":365}` |
| POST | `/push/stream-keys/revoke` | 撤销本人创建的 SSE 密钥，请求体为 `{"id":123}` |

创建接口返回 `token` 原文，仅这一次可见。长期有效的密钥可将 `expires_in_days` 设为 0，允许范围为 0 到 3650 天。
