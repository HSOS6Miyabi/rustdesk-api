# RustDesk API 数据库备份与恢复

适用范围：RustDesk API 开发分支，数据库迁移版本 **286**，使用 SQLite、MySQL 或 PostgreSQL。文中示例可以用于准备测试环境，正式恢复操作应在维护时间进行，并先核对自己的数据库地址、账户及备份路径。

需要同时保存 API 数据、RustDesk Server 身份密钥，以及客户端安装文件；只保存数据库可能导致安装文件记录与实际文件不一致。

## 操作前检查

1. 记录当前正在运行的 API 与 Server 版本。开发分支需要使用对应的 API 和 RustDesk Server 代码。
2. 查看配置文件中的 `gorm.type`，确认实际数据库为 `sqlite`、`mysql` 或 `postgresql`。
3. 准备只有管理员能读取的备份目录，确认可用空间大于数据库及安装文件的大小。
4. 将 `conf` 配置文件及实际设置的环境变量保存在安全位置。邮件服务启用时尤其需要保存 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`；密码、数据库连接信息和内部通信密钥不得上传 GitHub。
5. 按数据库类型备份；随后在单独的数据库或独立主机上恢复，检查记录后再进行正式升级。

## SQLite

SQLite 默认数据库为 API 工作目录下的 `data/rustdeskapi.db`，Docker 部署通常对应容器内 `/app/data/rustdeskapi.db`。

**应当先停止 API 服务，使数据库连接正常关闭，再复制数据库文件。** 如果服务由 systemd 管理，可以先执行：

```bash
sudo systemctl stop rustdesk-api
```

如果使用 Docker Compose：

```bash
docker compose stop rustdesk-api
```

确认 API 进程结束后，按实际目录复制 SQLite 文件：

```bash
mkdir -p "$HOME/rustdesk-backup"
cp -p ./data/rustdeskapi.db "$HOME/rustdesk-backup/rustdeskapi.db"
sha256sum "$HOME/rustdesk-backup/rustdeskapi.db"
```

以上路径是假设当前位于 API 工作目录的示例。若数据库挂载在其他位置，请使用实际宿主机路径。备份时若存在 `-wal` 或 `-shm` 文件，应确认服务已经完全退出，避免在未同步的数据上创建副本。

恢复演练时将备份文件复制到独立测试目录，再让测试版 API 使用该目录，检查管理员账户、设备数量、历史审计和版本迁移。执行前不要覆盖原有正式数据库。

## MySQL 8.4

先确认数据库名、服务器地址和具有备份权限的数据库账户。下面以数据库名 `rustdesk_api` 为例。执行时使用数据库客户端的安全密码提示。

```bash
mkdir -p "$HOME/rustdesk-backup"
mysqldump --host=127.0.0.1 --port=3306 --user=rustdesk \
  --single-transaction --no-tablespaces rustdesk_api \
  > "$HOME/rustdesk-backup/mysql-rustdesk.sql"
test -s "$HOME/rustdesk-backup/mysql-rustdesk.sql"
```

在独立测试环境中新建空数据库 `rustdesk_api_restore`，然后导入：

```bash
mysql --host=127.0.0.1 --user=rustdesk -e \
  "CREATE DATABASE rustdesk_api_restore CHARACTER SET utf8mb4;"
mysql --host=127.0.0.1 --user=rustdesk rustdesk_api_restore \
  < "$HOME/rustdesk-backup/mysql-rustdesk.sql"
```

备份账号需要读取目标表的权限；恢复账号需要创建表、索引及写入数据的权限。上述命令中的数据库名和账户应按实际情况修改。用于恢复测试的新数据库必须与正式数据库分开。

## PostgreSQL 16

使用 `pg_dump` 保存数据库，再在独立测试环境用 `pg_restore` 导入：

```bash
mkdir -p "$HOME/rustdesk-backup"
pg_dump --host=127.0.0.1 --port=5432 --username=rustdesk \
  --no-owner --no-acl --format=custom \
  --file="$HOME/rustdesk-backup/postgres-rustdesk.dump" rustdesk_api
test -s "$HOME/rustdesk-backup/postgres-rustdesk.dump"

createdb --host=127.0.0.1 --port=5432 --username=rustdesk rustdesk_api_restore
pg_restore --host=127.0.0.1 --port=5432 --username=rustdesk \
  --no-owner --no-acl --exit-on-error \
  --dbname=rustdesk_api_restore "$HOME/rustdesk-backup/postgres-rustdesk.dump"
```

上述恢复命令要求目标数据库为空，并且当前用户具备相应权限。如果使用外部数据库服务商，应按服务商要求提供 TLS 连接设置。

## 还需保存的文件

数据库备份之外，应当保存以下内容：

- **RustDesk Server 身份文件**：`hbbs` 使用的数据目录，尤其是 `id_ed25519` 与 `id_ed25519.pub`。更新服务后必须继续使用原有服务器身份。
- **客户端安装文件**：API 的 `data/client-builds`，或 `RUSTDESK_CLIENT_BUILD_DIR` 指定目录。这里保存已生成的官方安装文件与配置 ZIP。
- **配置与环境变量**：API 的实际 `conf` 目录、相关密钥、SMTP 加密密钥及数据库连接参数。
- **GeoLite2 文件**：如果启用了国家识别，保存 MMDB 文件或 CSV 数据所在目录，以便恢复后的中继选择保持一致。

务必限制备份目录的读取权限，避免将管理员密码、数据库账号、服务器私钥及用户数据放进公共代码仓库。

## 恢复完成后检查

请在测试环境执行以下检查，记录每项结果：

1. API 启动并能够正常连接恢复后的数据库；版本迁移成功。
2. 原有管理员和用户可登录；用户所属群组及设备归属保持不变。
3. 设备、安全策略、控制角色、地址簿、回收站和历史审计记录可以读取。
4. 邮件通知设置和客户端构建任务记录仍存在；安装文件可以下载，SHA-256 校验通过。
5. `hbbs` 的服务器公钥与备份前一致，`hbbr` 和 WebClient 能够启动。
6. 重启 API 与 Server 后，再次检查重要记录和连接状态。

自动化恢复测试由 `service/database_restore_test.go`、`service/database_multi_driver_test.go`、`scripts/test_database_restore.sh` 和 `.github/workflows/verify-database-upgrades.yml` 组成。GitHub Actions 使用一次性数据库进行测试，不会访问实际部署环境。

**正式升级前应另行完成真实备份恢复演练。** 完成后再安排新版本切换；当前开发分支尚未进入正式发布阶段。
