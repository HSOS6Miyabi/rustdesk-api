# 设置中心使用说明

系统管理员可以从管理控制台左侧的“系统管理 → 设置中心”查看和修改 RustDesk API 的全局配置。普通用户与受限管理员角色不能读取或修改这一页面。

## 查看设置

1. 使用系统管理员账户进入 `/_admin/`。
2. 打开“设置中心”，选择类别或输入配置名称、中文用途、环境变量名称进行搜索。
3. 每个项目会显示用途、当前运行值、实际来源、网页覆盖状态和保存位置。
4. 对密码与通信密钥，只显示是否已配置。需要修改时输入新值；留空保留原有覆盖值。
5. 上方的专项页面入口可打开 SMTP 邮件、通知订阅、日志保留、设备策略和中继节点等已有功能页面。

网页设置当前分为两类。数据库连接、监听端口、文件目录、密钥和运行附加服务等保存于 `data/settings-bootstrap.json`；一般应用设置保存在 `system_setting_overrides` 数据库表中。两个来源均按单个配置项启用覆盖。配置文件模板所展示的值与程序当前实际使用的值可能不同，请以页面中的“当前运行值”为准。

## 修改配置

1. 找到需要修改的项目，填写“网页覆盖值”。
2. 单击“保存并启用覆盖”，阅读确认说明后保存。
3. 页面提示“待重启生效”时，按照部署方式重新启动 API。
4. API 重新启动后重新打开该页面，确认“当前运行值”和“来源”已经变更。

设置中心修改全局配置后需要重新启动 API。网页保存成功后，正在运行的请求继续使用修改前的设置。涉及 API 与 hbbs 共同使用的 JWT 签名密钥、内部接口密钥或设备登记设置时，还需要检查配套 RustDesk Server 的环境变量是否一致。

例如，在服务器配置 `RUSTDESK_API_APP_REGISTER=true` 后，管理员仍可在网页中将 `app.register` 设置为 `false` 并启用覆盖。重新启动后注册功能关闭，来源显示为“网页设置”。

## 恢复使用环境变量或配置文件

选择某个已启用网页覆盖的项目，单击“关闭网页覆盖”。保存之后重新启动 API，程序就会按环境变量、配置文件和默认值的优先顺序重新计算该项设置。网页保存的旧值不会继续控制程序。

显式保存的布尔值 `false`、整数 `0` 和空字符串均为有效覆盖值；输入必须符合相应配置项的验证规则。

## 普通服务器重启

使用前先备份 `data/rustdeskapi.db`、`data/settings-bootstrap.json`、`conf/config.yaml` 及使用的加密密钥。在使用 systemd 管理 API 的情况下，通过实际服务名称执行：

```bash
sudo systemctl restart rustdesk-api
sudo systemctl status rustdesk-api
```

如果通过终端直接运行 API，应向进程发送正常停止信号，待程序退出后再重新启动同一个可执行文件。确认启动日志没有数据库迁移错误，再进入设置中心检查。

## Docker 重启

仓库中的 Docker Compose 示例将 `./data/rustdesk/api` 挂载到容器的 `/app/data`。配置文件 `settings-bootstrap.json` 与 SQLite 数据库都保存在这里。

```bash
docker compose up -d
docker compose restart rustdesk-api
docker compose logs --tail=100 rustdesk-api
```

重新创建容器之前，应确认数据卷仍挂载在 `/app/data`。修改数据库连接设置时，先备份原数据库并在独立环境中验证新数据库和权限。旧数据库中的用户、设备和地址簿不会因为修改连接地址而自动迁移。

## 多实例运行

当多个 API 实例共享同一个数据库时，应用设置覆盖记录由所有实例读取，但每个实例只有在重新启动后才会使用最新值。每台服务器还需分别保存自己的 `settings-bootstrap.json`。建议逐台重启并检查配置版本及服务状态。

多实例使用通知共享处理时，在所有 API 实例中启用 `RUSTDESK_API_SHARED_EVENT_WORKERS=1` 或相应网页设置，随后逐台重启。共享队列在处理事件前认领记录，并对意外退出的任务进行恢复。

设备安全策略具有独立的进程内缓存。多实例之间的缓存更新可能存在短暂延迟；严格连接访问控制还需要与 hbbs 的设置一起检查。

## 安全说明

- 不要将密钥填写在公开的聊天记录、工单或屏幕截图中。
- 启动配置文件只允许实际运行账户读取和写入。备份文件也应限制访问。
- 更换 `runtime.mail-encryption-key` 或 `runtime.push-encryption-key` 前，先核对已有 SMTP、TOTP 和推送渠道使用的加密凭据。修改旧密钥会影响已有数据的解密。
- 修改认证、数据库或内部通信设置后，检查当前管理员能否重新登录，并确认客户端仍能使用配套服务端。

## 接口

以下接口的路径均以 `/api/admin` 为前缀。它们需要账户登录令牌，并在服务端确认系统管理员身份。

| 接口 | 用途 |
| --- | --- |
| `GET /settings/schema` | 查看配置名称、用途、类型及环境变量 |
| `GET /settings/effective` | 查看当前运行值、来源、覆盖与待重启状态 |
| `POST /settings/validate` | 检查保存内容的格式 |
| `PATCH /settings/overrides` | 保存并启用某项或多项配置 |
| `POST /settings/overrides/reset` | 关闭指定配置的网页覆盖 |
| `GET /settings/history` | 查看最近的设置修改 |
| `GET /settings/overview` | 查看专项数据与待处理通知数量 |
| `GET /settings/outbox` | 查看处理失败的事件 |
| `POST /settings/outbox/retry` | 重新提交指定的失败事件 |

保存接口需要同时提交 `bootstrap_version`、`database_version` 和 `changes`。每次最多修改 50 项。同一请求只修改一种保存位置的设置；两个管理员使用旧版本号同时保存时，后一个请求会得到冲突提示，需要重新读取配置后再修改。

```json
{
  "bootstrap_version": 0,
  "database_version": 1,
  "changes": [
    {
      "key": "app.register",
      "enabled": true,
      "value": "false"
    }
  ]
}
```

读取敏感项目时不会返回密码或密钥原文。管理设置变更的历史记录只包含操作者、配置名称、启用状态和修改时间。

完整项目清单见 [全局配置参考](CONFIG_REFERENCE.md)。
## 多实例设备策略的更新延迟

当多台 API 实例共同读取同一个数据库时，可以在设置中心启用“多实例缓存同步”（`runtime.multi-instance-cache`），并重新启动每个实例。开启后，每台实例最多使用约 5 秒前的策略分配和通知订阅缓存；具体向客户端发送更新的时间还取决于客户端下一次心跳。没有启用此项时，缓存最长保留约 30 秒。

这项设置不会新增空闲时的数据库查询。对于需要在连接协商阶段立即判断的严格访问权限，应配置配套 Server 的连接许可接口，确保每次请求重新检查权限。
