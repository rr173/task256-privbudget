# task256-privbudget — 差分隐私统计预算组合验证服务

一款基于 Go 的后端服务：把数据集、统计机制（拉普拉斯/高斯/截断拉普拉斯）与发布任务登记入库后，按**顺序组合（sequential）/ 高级组合（advanced）/ RDP 组合（rdp）/ 数据集重叠图（graph）**四种规则实时评估各数据集的 ε/δ 预算消耗，检测超限路径，并支持撤销恢复与发布不可变预算快照。

## 业务闭环

1. **登记**：登记数据集版本（ε/δ 上限、父数据集血缘）与统计机制（机制类型、参数、作用的数据集集合）。
2. **验证**：机制通过验证（`draft → verified`）后才能参与发布；已撤销机制不可发布。
3. **发布评估**：创建发布批次，按所选规则计算该发布引起的预算消耗，允许/拒绝并记录超限路径。
4. **预算总览**：按规则评估全部数据集的实时预算状态（`releasable / overlimit`），数据集重叠图模式额外检测血缘环。
5. **撤销与恢复**：撤销已允许的发布，预算立即回退，数据集可恢复可发布。
6. **快照**：将当前预算状态冻结为不可变快照（`draft → published`），供审计追溯。

## 标准命令

```bash
# 构建 / 静态检查 / 测试
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...

# 启动服务（默认 :8080，SQLite 落盘 privbudget.db）
go run ./cmd/privbudget --addr :8080 --db privbudget.db

# 端到端自检（不启动长驻服务，验证持久化与重启恢复）
go run ./cmd/privbudget --smoke-test
```

## API 一览（前缀 `/api`，共 22 个端点）

| 能力 | 方法与路径 |
|---|---|
| 数据集登记/列表/详情/更新/封存 | `POST /api/datasets`、`GET /api/datasets`、`GET /api/datasets/{id}`、`PUT /api/datasets/{id}`、`POST /api/datasets/{id}/seal` |
| 机制登记/列表/详情/验证/撤销 | `POST /api/mechanisms`、`GET /api/mechanisms`、`GET /api/mechanisms/{id}`、`POST /api/mechanisms/{id}/verify`、`POST /api/mechanisms/{id}/revoke` |
| 发布创建/列表/详情/评估/撤销 | `POST /api/releases`、`GET /api/releases`、`GET /api/releases/{id}`、`POST /api/releases/{id}/evaluate`、`POST /api/releases/{id}/revoke` |
| 预算评估（GET/POST） | `GET /api/budget?rule=sequential\|advanced\|rdp\|graph`、`POST /api/budget/evaluate` |
| 快照创建/列表/详情/发布 | `POST /api/snapshots`、`GET /api/snapshots`、`GET /api/snapshots/{id}`、`POST /api/snapshots/{id}/publish?rule=...` |
| 自检 | `GET /api/selfcheck` |

## 目录结构

```
cmd/privbudget/        入口：--addr / --db / --smoke-test
internal/model/        实体与领域错误
internal/store/        SQLite 持久化（modernc.org/sqlite，CGO 无关）
internal/compose/      组合预算计算引擎（sequential/advanced/rdp/graph + 环检测）
internal/dataset/      数据集登记与校验
internal/mechanism/    机制登记与验证
internal/release/      发布评估与撤销
internal/snapshot/     快照冻结
internal/service/      编排层（写锁、预算状态刷新、自检）
internal/httpapi/      HTTP 层（/api 前缀）
```
