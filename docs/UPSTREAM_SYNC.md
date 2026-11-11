# 合并上游 RustDesk API 更新

本仓库源自 [lejianwen/rustdesk-api](https://github.com/lejianwen/rustdesk-api)。本分支保留原有 Go 模块路径 `github.com/lejianwen/rustdesk-api/v2`，以维持现有代码导入和构建方式。配套的 [RustDesk Server](https://github.com/HSOS6Miyabi/rustdesk-server) 是独立仓库，需要分别检查和更新。

## 开始之前

请先阅读 [分支功能说明](FORK_CHANGELOG.md)、[兼容版本](COMPATIBILITY.md) 和 [测试说明](TESTING.md)。检查是否有正在进行的数据库结构修改和未合并的功能分支，保留它们的现有状态。

在 2026 年 11 月 11 日首次审查时，上游 `master` 的提交为 `c5687e150615bfb00d9d105f0883a2802750a8ad`，本仓库 `master` 为 `f81142412366e581ea22f548567405611138cb5c`。当时本分支比上游多 520 个提交，没有缺少的上游提交。此信息用于记录历史检查结果；每次更新前应重新查询。

## 初次添加上游

```bash
git clone https://github.com/HSOS6Miyabi/rustdesk-api.git
cd rustdesk-api
git remote -v
git remote add upstream https://github.com/lejianwen/rustdesk-api.git
git fetch upstream --tags
git fetch origin --tags
git switch master
git pull --ff-only origin master
```

如已有 `upstream` 远程仓库，先用 `git remote -v` 核实地址，无需重复执行 `git remote add`。

## 检查上游变化

```bash
git fetch upstream --tags
git fetch origin --tags
git rev-list --left-right --count upstream/master...master
git log --oneline master..upstream/master
git diff --stat master..upstream/master
```

第一条数字表示上游独有的提交数，第二条表示本分支独有的提交数。若第一条为零，本次没有需要合并的上游提交。

## 建立独立合并分支

```bash
git switch master
git pull --ff-only origin master
git switch -c maintenance/upstream-sync-YYYYMMDD
git merge --no-commit --no-ff upstream/master
```

如出现冲突，执行 `git status` 查看文件；逐项阅读双方更改，确定现有用户、设备、地址簿、客户端协议和数据库记录仍然兼容。不要使用 `git reset --hard upstream/master` 覆盖本项目的附加功能。

重点核对 `cmd/apimain.go`、`model/`、`service/`、`http/`、`config/`、`resources/admin/`、`resources/web/js/`、Docker 构建文件和自动化测试。自动合并成功的文件也可能存在行为差异，需要检查关键业务代码。

## 合并后的检查

```bash
git diff --check
go test ./... -count=1
go test -race ./service ./lib/cache ./http/... -count=1
go build -o /tmp/rustdesk-api ./cmd/apimain.go
node --check resources/admin/app.js
node --test resources/admin/app.test.cjs
```

还需运行 GitHub Actions 中的 SQLite、MySQL、PostgreSQL 迁移与恢复测试、浏览器交互测试以及涉及 hbbs/hbbr 的实际协议测试。程序需要同时支持普通服务器安装和 Docker 部署。使用正式数据库升级之前应在独立环境恢复备份，并先完成测试。

所有文件检查完成后，执行 `git add`、`git commit`，并创建拉取请求，逐项说明上游提交范围、涉及功能、数据库变化、测试结果和恢复方法。通过审核之后再合并到 `master`。

## 正式版本发布

普通提交和合并拉取请求不会自行创建版本标签。发布时要明确 API 版本、数据库迁移版本、配套 Server 版本以及适用客户端版本。只有通过测试并获得明确发布授权之后，才按照现有发布工作流创建 GitHub Release 与 Docker 镜像。

每次合并上游更新后，都更新 [分支功能说明](FORK_CHANGELOG.md)、[兼容版本](COMPATIBILITY.md) 与 [项目维护](MAINTENANCE.md)。如上游涉及服务端协议，另在 RustDesk Server 仓库执行对应代码审查与测试。