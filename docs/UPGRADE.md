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

## 多中继服务器

API 数据库自动增加 `relay_nodes` 表，新增国家代码字段后的迁移版本为 277。已有服务器和设备记录不会更改。

管理员可以在 Web 控制台“中继节点管理”中增删节点。如果不需要集中管理多个中继，可以不配置任何节点，继续使用原有 `hbbs -r` 参数。

启用自动同步时，在 `hbbs` 环境中配置：

```bash
export RUSTDESK_API_RELAY_NODES_URL="http://127.0.0.1:21114/internal/v1/relay/nodes"
export RUSTDESK_API_INTERNAL_KEY="与 API 一致的内部访问密钥"
export RUSTDESK_API_RELAY_SYNC_SECONDS=30
```

建议将 API 内部接口放在受保护的网络中。多台服务器之间通信应使用 HTTPS。默认 30 秒同步一次，允许配置 10 至 3600 秒。动态配置全部清除后将恢复原有中继地址设置。

## GeoLite2 Country CSV 地理位置识别

仅使用国家级节点偏好时需要此项设置。获取官方 [GeoLite2 Country CSV](https://dev.maxmind.com/geoip/docs/databases/city-and-country/)，解压后将以下三个文件放在 hbbs 可读取的同一目录：

- `GeoLite2-Country-Locations-en.csv`
- `GeoLite2-Country-Blocks-IPv4.csv`
- `GeoLite2-Country-Blocks-IPv6.csv`

为 hbbs 设置：

```bash
export RUSTDESK_GEOIP_CSV_DIR="/opt/rustdesk/geoip"
```

重启 hbbs 后自动读取数据。更新 GeoLite2 文件后，可通过原有的本地服务端控制接口发送 `reload-geo`，无需重启服务端。

没有提供 Country CSV 时，系统保持原有 CIDR 规则和中继选择功能。GeoLite2 Country CSV 数据的下载及使用应遵守 MaxMind 的授权要求。

## 远程控制角色

API 数据库迁移版本为 278，自动增加 `control_roles` 与 `control_role_assignments`。原有设备和账户默认无控制角色分配，会继续使用现有远程会话权限设置。

若需要让 hbbs 在连接协商时下发控制角色，增加以下环境变量：

```bash
export RUSTDESK_API_CONTROL_ROLE_URL="http://127.0.0.1:21114/internal/v1/control/permissions"
export RUSTDESK_API_INTERNAL_KEY="与 API 配置相同的内部通信密钥"
export RUSTDESK_API_CONTROL_CACHE_SECONDS=5
```

使用 Docker 或不同服务器时，需要把同步 URL 换成 hbbs 实际可访问的 API 内部地址。可在 Web 管理界面“远程控制角色”和“远程控制角色分配”中创建角色并指定应用范围。

## 管理员分级权限

数据库迁移版本为 279，新增 `admin_roles` 和 `admin_role_assignments`。原有系统管理员保留 `is_admin=true` 和全部管理功能，普通用户不会自动获得管理权限。

管理员在“管理员角色”中创建角色、勾选操作权限，再通过“管理员角色分配”将角色分配给指定成员。

如果需要限制查看范围，选择“指定用户组”或“指定用户”，并填写对应编号。限定范围目前适用于用户目录、设备目录的查看操作。需要编辑或删除等操作时选择全局角色并勾选相应权限。

成员丢失双重认证凭据时，具备相应权限的管理员可以在“用户账户”中选择“重置双重认证”，系统会清除原有第二因素及登录会话。
