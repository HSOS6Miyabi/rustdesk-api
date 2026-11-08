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
