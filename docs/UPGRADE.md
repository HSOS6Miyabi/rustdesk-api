# 更新现有安装

升级前请备份 API 数据库以及 RustDesk 服务端设备注册数据库。保存 ID 服务器身份密钥、公钥及相关环境配置。

## 数据库更新

API 启动时自动为以下内容创建或增加字段：

- `device_deployments`：严格登记模式的设备信息。
- `peers.note`：设备备注。
- `address_books.note`：地址簿条目备注。

原有账户、设备、地址簿、群组、审计及策略记录保持不变。SQLite、MySQL 和 PostgreSQL 都使用 GORM 迁移结构。

## 默认配置

`app.require-device-deployment` 默认值为 `false`。未启用时设备注册方式与此前相同，客户端 `--deploy` 得到 `NOT_ENABLED`。

如需启用严格登记：

API 与 Server 都可以通过相同的环境变量设置：

```bash
export REQUIRE_DEVICE_DEPLOYMENT=Y
export RUSTDESK_API_INTERNAL_KEY="替换为随机且保密的内部通信密钥"
```

或者在 API 中使用 YAML 配置：

```yaml
app:
  require-device-deployment: true
  internal-key: "替换为随机且保密的内部通信密钥"
```

RustDesk Server 设置：

```bash
export REQUIRE_DEVICE_DEPLOYMENT=Y
export RUSTDESK_API_INTERNAL_KEY="与 API 一致的内部通信密钥"
export RUSTDESK_API_INTERNAL_URL="http://127.0.0.1:21114/internal/v1/device/authorize"
```

严格登记模式只应在 API 可访问且两边配置完成后启用。正式升级前使用测试设备验证注册和登录。

在独立主机部署时，应更换内部接口地址并限制访问来源。

## 发布

普通提交和版本标签不会自动创建 Release。完成里程碑后在标签上手动运行 GitHub Actions，并明确启用 `milestone_release`。

## 新增管理数据

新版本通过 GORM 自动建立以下表：

- `device_policy_user_assignments`：用户策略分配。
- `device_policy_group_assignments`：设备组策略分配。
- `device_active_connections`：最近上报的活动连接。
- `device_disconnect_requests`：管理员发出的结束连接请求及处理状态。
- `api_tokens`：独立管理令牌及撤销记录。

`device_policies.status` 用于启用或停用策略。历史策略默认启用，原有直接设备分配保持有效。

普通账户会话令牌和 API 管理令牌分别保存。升级后可在“API 管理令牌”页面按需创建新的自动化令牌。已有账户登录方式不改变。

## 地址簿数据恢复和邮件功能

数据库自动为 `address_books` 增加 `deleted_at` 字段。删除单个条目时记录删除时间；旧数据继续正常显示。删除整个地址簿分类会清除其所有相关条目，恢复操作需在分类仍然存在时进行。

新增表：`smtp_settings`、`password_reset_tokens`、`user_invitations`。这些表不会修改原有账户密码和群组关系。

若使用 SMTP 账号密码，启动 API 前设置：

```bash
export RUSTDESK_API_MAIL_ENCRYPTION_KEY="更换为长期保存的随机密钥，至少16个字符"
```

同一安装应长期保留此值，备份恢复时需要一起恢复。SMTP 密码仅以加密内容保存，重新配置密码时才会替换。邮件设置的 `base_url` 需要填写用户能实际访问的 Web 控制台地址，例如 `https://remote.example.com`。

忘记密码和邀请账户依靠 SMTP 发送邮件。未设置 SMTP 时，原有用户仍可通过已启用的登录方式访问系统。

## 连接记录备注

`audit_conns` 新增 `note` 和 `note_updated_by`。历史记录的备注默认为空，原有连接时间、身份信息及导出功能保留。管理员可在“全部连接记录”添加和查询备注。

## TOTP 双重认证

数据库自动新增 `user_totps`、`totp_recovery_codes`、`totp_login_challenges`。原有用户默认不启用 TOTP，已安装的普通客户端可继续登录。

如需启用身份验证器，在服务器设置并保存 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`，例如：

```bash
export RUSTDESK_API_MAIL_ENCRYPTION_KEY="替换成至少32个随机字符的长期加密密钥"
```

该密钥同时用于加密 SMTP 密码和 TOTP 密钥。即使暂时不使用 SMTP，TOTP 仍需要此环境变量。备份数据库时应同时保存密钥，避免升级后无法验证已有的身份验证器。

用户在 Web 控制台“个人设置”中启用 TOTP、输入验证码并妥善保存六个恢复码。

## 可选邮箱验证码登录

数据库增加 `email_login_settings`、`email_code_challenges`，迁移版本为 275。原有账户默认不启用邮箱验证码，现有密码登录保持有效。

管理员在“SMTP 邮件服务”中设置邮件服务器，普通用户在“个人设置”中确认自己的账户邮箱，并选择“发送邮箱验证码并启用”。需要真实发送邮件才能完成绑定。

启用 TOTP 的用户优先使用 TOTP。若用户更改了账户邮箱，需要在个人设置重新验证该邮箱后才能继续使用邮箱验证码登录。
