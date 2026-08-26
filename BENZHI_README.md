基于 Go 实现的差分隐私统计预算组合验证 Web 项目，一款后端服务，完成数据集预算登记、机制验证、多规则组合预算评估与不可变预算快照发布。

# BENZHI 评测说明

本项目为纯后端 Go 服务，对外暴露 `/api` 前缀的 HTTP 接口，使用 SQLite 持久化，
支持进程关闭后重新打开同一数据库恢复全部业务数据。

## 标准命令

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/privbudget --smoke-test
go run ./cmd/privbudget --addr :8080 --db privbudget.db
```

- `--addr`：HTTP 监听地址，默认 `:8080`
- `--db`：SQLite 数据库文件路径，默认 `privbudget.db`
- `--smoke-test`：不常驻；跑完端到端场景后关闭并重新打开数据库，退出码 0 表示通过

## 冒烟自测契约（--smoke-test）

创建临时数据库 → 登记共享同一数据集的两个机制并验证 → 顺序评估两个发布使组合超限 → 关闭并重新打开数据库校验超限状态恢复 → 撤销其中一个发布后恢复可发布 → 在限额内发布预算快照 → 再次关闭重开校验快照保持 published 且冻结条目不可变 → 运行自检后退出 0。

## Docker 构建与双架构验证

`Dockerfile` 与 `benzhi.Dockerfile` 内容完全一致。使用项目提供的
`build_benzhi_docker.sh` 构建评测镜像；Dockerfile 不声明端口，服务监听地址由
运行参数 `--addr` 指定。

```bash
./build_benzhi_docker.sh task256-privbudget:amd64 linux/amd64
docker run --rm task256-privbudget:amd64 --smoke-test

./build_benzhi_docker.sh task256-privbudget:arm64 linux/arm64
docker run --rm task256-privbudget:arm64 --smoke-test

docker run --rm -P task256-privbudget:amd64 --addr :8080 --db ./app.db
```

## 核心 API（`/api` 前缀）

- 数据集：`POST /api/datasets`、`GET /api/datasets`、`GET /api/datasets/{id}`、`PUT /api/datasets/{id}`、`POST /api/datasets/{id}/seal`
- 机制：`POST /api/mechanisms`、`GET /api/mechanisms`、`GET /api/mechanisms/{id}`、`POST /api/mechanisms/{id}/verify`、`POST /api/mechanisms/{id}/revoke`
- 发布：`POST /api/releases`、`GET /api/releases`、`GET /api/releases/{id}`、`POST /api/releases/{id}/evaluate`、`POST /api/releases/{id}/revoke`
- 预算：`GET /api/budget?rule=sequential|advanced|rdp|graph`、`POST /api/budget/evaluate`
- 快照：`POST /api/snapshots`、`GET /api/snapshots`、`GET /api/snapshots/{id}`、`POST /api/snapshots/{id}/publish?rule=...`
- 自检：`GET /api/selfcheck`

## 环境与组件

- Go 1.26.3（GOTOOLCHAIN=local，CGO_ENABLED=0）
- SQLite 3.46.1（modernc.org/sqlite v1.52.0，CGO 无关）
