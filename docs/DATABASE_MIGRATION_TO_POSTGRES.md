# 将 MySQL 或 SQLite 数据迁移到 PostgreSQL

本说明适用于 RustDesk API 使用的 MySQL、MariaDB 和 SQLite 数据库。仓库中的 `scripts/convert_to_postgres.py` 可以把 MySQL/MariaDB 的 SQL 导出文件或 SQLite 数据库文件转换成 PostgreSQL 可以执行的 SQL。脚本只读取来源文件，生成新的输出文件，不会连接或修改正在运行的数据库。运行脚本需要 Python 3.10 或更新版本，不需要安装第三方 Python 软件包。

**迁移前请停止 RustDesk API 以及所有会向原数据库写入数据的服务。** 如果迁移期间仍有新记录写入原数据库，转换结果将缺少这些新记录。正式切换之前，请对原数据库及配套文件分别进行备份，并在独立的测试数据库中检查转换结果。

## 一、准备 PostgreSQL 数据库

建议使用 PostgreSQL 16 或更新版本，并为迁移建立一个全新的空数据库。下面以数据库名称 `rustdesk_api` 和 PostgreSQL 用户 `rustdesk` 为例。数据库用户需要具备在目标数据库内创建数据表、索引及序列的权限。

```bash
createdb --host=127.0.0.1 --port=5432 --username=rustdesk rustdesk_api
```

如果准备使用已有数据库，请先检查是否存在同名数据表。转换后的 SQL 在同名表已经存在时会报错，以避免覆盖已有数据。

## 二、从 MySQL 或 MariaDB 迁移

先使用数据库客户端导出数据库的**表结构和全部数据**。导出时应选择 UTF-8 字符编码。下面使用 `mysqldump` 举例：

```bash
mysqldump --host=127.0.0.1 --port=3306 --user=rustdesk \
  --single-transaction --hex-blob --no-tablespaces --default-character-set=utf8mb4 \
  rustdesk_api > mysql-backup.sql
```

如果使用 Navicat，请同时导出表结构与数据。只包含导出说明、`SET` 语句或注释的文件无法迁移，程序会报告缺少 `CREATE TABLE`。

在仓库根目录运行：

```bash
python3 scripts/convert_to_postgres.py \
  --source mysql \
  --input mysql-backup.sql \
  --output postgres-import.sql
```

完成后会打印转换成功的数据表数量、记录数量及输出路径。文件包含建表语句、数据、普通索引、唯一索引、外键和自增序列的调整语句。

## 三、从 SQLite 迁移

RustDesk API 默认使用 `data/rustdeskapi.db`。停止 API 服务并确认数据库已经正常关闭后，复制数据库文件，再对副本执行转换。

```bash
cp -p data/rustdeskapi.db sqlite-backup.db
python3 scripts/convert_to_postgres.py \
  --source sqlite \
  --input sqlite-backup.db \
  --output postgres-import.sql
```

SQLite 转换会保留表中记录、主键、常规索引、唯一索引及外键声明。二进制数据会转换成 PostgreSQL 的 `BYTEA` 表达式。

## 四、将 SQL 导入 PostgreSQL

请先在独立的测试数据库执行一次导入：

```bash
psql --host=127.0.0.1 --port=5432 --username=rustdesk \
  --dbname=rustdesk_api --set=ON_ERROR_STOP=1 --file=postgres-import.sql
```

脚本输出的 SQL 使用 `BEGIN` 和 `COMMIT` 包含整个迁移过程。若某条 SQL 执行失败，PostgreSQL 会中止该事务；执行完成后，务必检查 `psql` 的退出状态。

检查用户、设备和数据库版本记录的数量：

```sql
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM peers;
SELECT COUNT(*) FROM versions;
SELECT MAX(version) FROM versions;
```

请将这些结果与来源数据库的查询结果进行比较。还应检查地址簿、设备分组、安全策略、历史审计记录以及设置中心中的配置。表名应以实际数据库中的名称为准。

## 五、修改 RustDesk API 配置

迁移验证成功后，停止旧服务，将 `conf/config.yaml` 中 `gorm.type` 设置为 `postgresql`，并填写对应的 `postgresql` 连接参数。可以通过环境变量填写密码等敏感信息。若此前通过设置中心保存了数据库连接覆盖设置，应先检查 `data/settings-bootstrap.json`，以确认重启后实际使用的连接参数。

启动 RustDesk API，确认程序能够连接 PostgreSQL，并检查日志中是否存在数据库迁移错误。随后登录管理页面，核对用户、设备、地址簿及历史记录。切换期间保留原数据库及备份，确认服务运行正常后再处理旧数据库。

数据库备份及其他必须保存的文件见 [数据库备份与恢复](DATABASE_BACKUP_RESTORE.md)。

## 六、适用范围与注意事项

- 脚本仅接受 **MySQL/MariaDB SQL 导出文件**和 **SQLite 数据库文件**作为输入；不直接连接 MySQL 或 PostgreSQL。MySQL 数据应先使用 `mysqldump` 或 Navicat 导出。
- 请使用空的 PostgreSQL 数据库，导入前不要启动 RustDesk API 让程序自动创建表。
- 支持常见数值、字符、时间、布尔和二进制字段；同时转换主键、普通索引、唯一索引和常规外键。MySQL 的无符号整数使用能够容纳相应范围的 PostgreSQL 类型。
- MySQL/MariaDB 的字符排序规则与 PostgreSQL 可能不同，特别是大小写敏感性。迁移后请检查用户名等唯一字段的实际比较行为。
- 某些数据库特有功能，例如存储过程、触发器、视图、生成列、部分索引，以及 MySQL 的 `ON UPDATE` 自动更新时间字段，没有直接等价的转换。遇到这类结构时，脚本会停止并报告原因，避免静默遗漏。
- MySQL 允许的零日期在 PostgreSQL 中无效。发现此类数据时，需要先在来源数据库中核实和修正，随后重新导出。
- 超出 PostgreSQL `BIGINT` 范围的 MySQL 无符号自增编号需要单独处理序列范围；历史编号本身会保存在 `NUMERIC(20,0)` 字段中。
- PostgreSQL 生成的表结构应与当前 RustDesk API 版本兼容。切换前应使用相同版本的 RustDesk API 在测试环境进行完整的启动和功能检查。
- 转换后的 SQL 含有用户账户、设备信息及可能的敏感配置。请将文件保存在权限受限的目录，不要提交到 GitHub，也不要复制到公开的故障报告中。

## 七、执行测试

```bash
python3 -m unittest discover -s tests -p 'test_convert_to_postgres.py' -v
```

该测试使用自行构造的样例数据，检查 MySQL 语句解析、SQLite 文件读取、索引及外键生成、字符与二进制数据转换，以及遇到不支持的语法时是否安全终止。仓库的 PostgreSQL 转换测试工作流还会将生成的 SQL 导入独立的 PostgreSQL 测试数据库。