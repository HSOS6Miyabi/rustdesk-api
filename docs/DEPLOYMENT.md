# 部署与配置

## 1. 运行 API

API 默认监听 `21114`。在 `conf/config.yaml` 中修改 `gin.api-addr`，可以指定其他监听地址。SQLite 默认保存在 `data/rustdeskapi.db`，请将整个 `data` 目录持久保存。

执行：

```bash
go mod download
go build -o apimain ./cmd/apimain.go
mkdir -p data runtime
./apimain
```

首次运行后，打开 `http://服务器地址:21114/_admin/`，使用 `admin` 和启动日志中的初始密码登录。建议在反向代理服务器上配置 HTTPS。

## 2. 配置 RustDesk 服务器

在 `conf/config.yaml` 中填写：

```yaml
rustdesk:
  id-server: "desk.example.com:21116"
  relay-server: "desk.example.com:21117"
  api-server: "https://api.example.com"
  key: "这里填写服务器公钥"
  personal: 1
jwt:
  key: "请填写至少32位随机字符串"
```

也可使用环境变量 `RUSTDESK_API_RUSTDESK_ID_SERVER`、`RUSTDESK_API_RUSTDESK_RELAY_SERVER`、`RUSTDESK_API_RUSTDESK_API_SERVER` 和 `RUSTDESK_API_JWT_KEY`。

使用配套 RustDesk Server 时，将相同的 `RUSTDESK_API_JWT_KEY` 配置给服务端，使服务器能够验证用户登录令牌。可通过 `MUST_LOGIN=Y` 要求设备连接前登录。默认 `MUST_LOGIN=N`。

## 3. 为 WebClient v2 配置 HTTPS 和 WebSocket

在浏览器中打开 `https://api.example.com/webclient2/` 后，可以使用设备 ID 连接 RustDesk 服务。网页登录后，页面会自动读取 ID 服务器、公钥和共享设备列表。

RustDesk 服务端使用 `21118` 端口提供 ID 服务器的 WebSocket，使用 `21119` 端口提供中继 WebSocket。浏览器通过 HTTPS 访问 API 时，WebSocket 也必须使用有效的 TLS 证书。可以在 Nginx 上设置两个 HTTPS WebSocket 代理入口：

```nginx
server {
    listen 443 ssl;
    server_name desk.example.com;
    ssl_certificate /etc/letsencrypt/live/desk.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/desk.example.com/privkey.pem;

    location = /ws/id {
        proxy_pass http://127.0.0.1:21118/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 3600s;
    }

    location = /ws/relay {
        proxy_pass http://127.0.0.1:21119/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 3600s;
    }
}
```

如果 RustDesk 服务端在另一台机器上运行，请将 `127.0.0.1` 替换为其内网地址，并确保 WebSocket 代理能够访问端口 `21118` 和 `21119`。证书路径应根据实际的域名与证书位置填写。

为保证来源 IP 的准确性，应限制服务器的 `21118` 和 `21119` 端口，仅允许受信任的反向代理访问。反向代理必须自行填写 `X-Real-IP` 请求头，避免使用外部请求中提供的值。

在 API 的 `conf/config.yaml` 中设置：

```yaml
rustdesk:
  id-server: "desk.example.com:21116"
  relay-server: "desk.example.com:21117"
  api-server: "https://api.example.com"
  ws-host: "wss://desk.example.com"
```

`ws-host` 用于生成 `wss://desk.example.com/ws/id` 和 `wss://desk.example.com/ws/relay` 两个连接地址。修改配置后重新启动 API。已有的设备 ID 与远程连接密码不受影响。

从源代码部署时，需要先构建网页连接组件：

```bash
cd resources/web/js
npm ci
npm run build
cd ../../..
```

GitHub Release 和 GHCR 镜像会自动包含已构建的文件。

## 4. Docker

使用仓库自带的 Dockerfile：

```bash
docker build -t rustdesk-api .
docker run --name rustdesk-api -d \
  -p 21114:21114 \
  -e RUSTDESK_API_RUSTDESK_API_SERVER=https://api.example.com \
  -v "$PWD/data:/app/data" \
  --restart unless-stopped rustdesk-api
```

也可从 GHCR 获取发行镜像。管理员密码和数据库保存在挂载的数据目录中。升级容器时继续挂载同一个目录。

## 5. 群组和联系人

管理员在“用户分组”页面创建普通组、共享组或公共锚点组。创建公共锚点组后，打开“公共成员”选择需要对组员公开的账户。用户可通过主要分组或附加分组加入公共锚点组。

“批量管理成员”支持添加、移除及替换，成员变更后会影响下一次客户端请求。属于公共锚点组的管理员在 RustDesk 客户端也只看到自己和选中的公共成员及其设备。后台的管理员设备管理页面仍可查看全部设备。

任何用户都可以在“联系人”页面填写其他用户的用户名。添加后，自己的设备会展示在对方的“共享设备”页面。对方不需要接受邀请；撤销联系人后，相关设备从共享列表消失。通过其他方式得到设备 ID 的用户仍可按照原有 RustDesk 连接规则发起连接。

## 6. 个人群组和审计记录

所有用户都可以在“我的群组”页面创建群组，管理群组名称、成员和公共锚点成员。个人群组的创建者会自动加入该群组，并保留原来的主要分组。群组所有者只能编辑自己创建的群组，管理员可在独立的“用户分组”页面管理所有群组。

添加成员时可以填写已知用户的完整用户名。为防止用户通过群组管理接口查看其他群组的成员，个人群组的查找功能仅按完整用户名查询。普通用户无法通过接口修改管理员的群组成员关系。

“我的连接记录”展示用户主动连接远程设备和其他人连接自己设备的日志。“我的文件传输”展示本人设备参与的文件传输。历史日志会利用设备 ID 查询现有归属信息；新日志保存连接双方的信息，在设备更换归属后仍能显示原有记录。

## 7. OIDC / SSO

管理员在“OIDC / SSO”页面设置身份认证服务的 Client ID、Client Secret、Issuer、Scopes，并在“自动加入的分组 ID”中填写用逗号分隔的群组编号，例如 `2,3`。

每次通过该身份认证服务成功登录时，系统会检查并添加相应成员关系，且不会重复添加。一个用户可以属于多个分组。管理员修改自动加入的群组后，新规则会在下次登录时应用。

## 8. 地址簿

控制台“地址簿”页面有“设备条目”和“地址簿分类”两个视图。每个分类可以包含多个设备条目。已有数据库中的分类及条目会继续保留，RustDesk 客户端原有的地址簿接口也可继续使用。

## 9. 升级与备份

升级前先备份数据库。SQLite 使用 `data/rustdeskapi.db`；MySQL 和 PostgreSQL 请使用相应数据库管理工具。更新代码或镜像后重新启动 API。程序启动时自动检查数据库结构并添加新字段和表。


## 10. WebRTC 连接

本项目的 RustDesk Server 可传递 RustDesk 1.5.x 的 SDP 和 ICE 候选地址信息，并保留原有 TCP、UDP 和 WebSocket 中继连接方式。

桌面客户端需要在本地启用 `enable-webrtc=Y`。使用 `/webclient2/` 时，可勾选“优先通过 WebRTC 直接连接”。浏览器会优先尝试 WebRTC，失败后重新尝试 WebSocket 中继。浏览器需要支持 `RTCPeerConnection`，建议使用 HTTPS；网络需要允许必要的 UDP 通信。浏览器端会检查被控设备签名的 DTLS 证书指纹，以验证连接身份。

## 11. 设备安全策略

管理员在“设备安全策略”中创建策略，再在“设备策略分配”中填写设备 ID 和策略编号。RustDesk 1.5.x 客户端通过 `/api/heartbeat` 自动接收设备 ID 白名单、IP 白名单及可信设备有效期。

设备需要先通过 `/api/sysinfo` 向服务器报告 ID 和 UUID。API 在心跳期间检查二者一致后才会返回对应设备的安全策略。管理员修改或撤销分配后，客户端会在之后的心跳中更新设置。

WebRTC 设置保存在官方客户端本地。策略中记录的 WebRTC 选项可供管理员参考，需要在客户端中单独启用。

## 12. 日志管理

普通用户可以查看“我的安全告警”，管理员可以查看全部安全告警、管理操作记录并下载 CSV。管理员也可以在“日志保留设置”中分别设置连接、文件传输、告警、登录和管理操作记录的保留期限。

默认永久保留。启用自动清理后，服务启动时和每隔 24 小时检查并删除超过期限的记录。建议在调整期限前备份数据库。

## 12. 设备部署、管理令牌与活动连接

小型团队可以继续使用默认的设备注册方式，无须执行额外的部署步骤。希望限制新设备登记时，按照 [升级说明](UPGRADE.md) 启用严格设备登记，并确保 API 与 `hbbs` 配置一致。

管理员在“API 管理令牌”中创建适当权限的令牌，便于在已安装的 RustDesk 客户端执行 `--assign` 或 `--deploy`。令牌可随时在控制台撤销，不需要把管理员登录密码写入部署脚本。

“设备安全策略”页面管理策略，“设备策略分配”管理单台设备，“用户策略分配”和“设备组策略分配”管理批量设备。客户端会根据优先级取得生效策略。

“活动远程连接”仅显示近期通过心跳上报的会话。管理员可以请求结束某个连接，并在“结束连接记录”查看客户端通知和结束结果。

上述功能使用同一套 API 和数据库，普通服务器与 Docker 部署都可以使用。

## 13. 与其他用户共享地址簿

普通用户可以在控制台“我的地址簿共享规则”页面，选择自己创建的地址簿，指定分享给某位用户、某个群组或所有用户，并设置只读、读写、完全控制三种权限。

同一用户加入多个群组时，所有群组的共享权限都会参与计算。用户加入或退出群组后，下次从客户端读取共享地址簿时就会取得新的结果。

如果同时设置了个人规则、群组规则和所有用户规则，个人规则优先，其次为群组规则，最后为所有用户规则。公共锚点组的成员可见范围继续按原有设置生效。

## 14. 地址簿 CSV 与回收站

在“地址簿”页面选择“导入 CSV”或“导出 CSV”可以批量维护设备，数据格式参见 [API 使用说明](API.md)。普通用户操作自己的地址簿，系统管理员可以指定需要维护的账户。

误删的设备条目可进入“地址簿回收站”恢复；恢复后原有连接信息继续保留。删除整个地址簿分类时会永久删除该分类相关条目，执行前建议导出一份 CSV，并确保数据库已有备份。

## 15. SMTP 邮件服务、密码恢复与邀请成员

在服务运行环境配置 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`。如果使用需要密码的 SMTP 服务，建议生成至少 32 个随机字符，并长期保存，不能在升级时随意改变。

管理员在“SMTP 邮件服务”页面填写主机、端口、连接方式、账户、发件邮箱以及用户能够访问的控制台 URL。保存设置后，向自己的邮箱发送测试邮件检查是否能够正常接收。

SMTP 可用后，登录页面提供“忘记密码”功能；管理员还可以在“成员邮件邀请”中指定邮箱与用户组，向新成员发送注册链接。默认不会强制现有用户启用这些功能。

## 16. 多中继节点

在“中继节点管理”中输入中继服务器的公开地址，按需填写优先使用该节点的 IP 网络范围和优先级。已启用的节点会通过 API 内部接口供 `hbbs` 查询。

如果 API 与 `hbbs` 在同一台服务器，可设置：

```bash
export RUSTDESK_API_RELAY_NODES_URL="http://127.0.0.1:21114/internal/v1/relay/nodes"
export RUSTDESK_API_INTERNAL_KEY="与 API 配置一致的密钥"
```

分离部署时将同步地址换成实际的内部 API 地址，并使用受信任的网络或 HTTPS。默认约 30 秒同步一次。

未配置上述同步地址时，仍使用原有中继地址及配置方式，不影响单中继部署。
