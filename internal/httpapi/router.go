// Package httpapi 暴露 /api 前缀的 HTTP 接口，覆盖数据集、机制、发布、
// 预算评估、快照与自检。路由使用 Go 1.22+ 的 ServeMux 路径通配符。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/service"
)

// Handler 持有应用服务并注册所有路由。
type Handler struct {
	app *service.App
}

// NewHandler 构造 HTTP 处理器。
func NewHandler(app *service.App) *Handler { return &Handler{app: app} }

// Routes 返回配置好的 http.Handler。
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/datasets", h.registerDataset)
	mux.HandleFunc("GET /api/datasets", h.listDatasets)
	mux.HandleFunc("GET /api/datasets/{id}", h.getDataset)
	mux.HandleFunc("PUT /api/datasets/{id}", h.updateDataset)
	mux.HandleFunc("POST /api/datasets/{id}/seal", h.sealDataset)

	mux.HandleFunc("POST /api/mechanisms", h.registerMechanism)
	mux.HandleFunc("GET /api/mechanisms", h.listMechanisms)
	mux.HandleFunc("GET /api/mechanisms/{id}", h.getMechanism)
	mux.HandleFunc("POST /api/mechanisms/{id}/verify", h.verifyMechanism)
	mux.HandleFunc("POST /api/mechanisms/{id}/revoke", h.revokeMechanism)

	mux.HandleFunc("POST /api/releases", h.createRelease)
	mux.HandleFunc("GET /api/releases", h.listReleases)
	mux.HandleFunc("GET /api/releases/{id}", h.getRelease)
	mux.HandleFunc("POST /api/releases/{id}/evaluate", h.evaluateRelease)
	mux.HandleFunc("POST /api/releases/{id}/revoke", h.revokeRelease)

	mux.HandleFunc("GET /api/budget", h.evaluateBudget)
	mux.HandleFunc("POST /api/budget/evaluate", h.evaluateBudgetPost)

	mux.HandleFunc("POST /api/snapshots", h.createSnapshot)
	mux.HandleFunc("GET /api/snapshots", h.listSnapshots)
	mux.HandleFunc("GET /api/snapshots/{id}", h.getSnapshot)
	mux.HandleFunc("POST /api/snapshots/{id}/publish", h.publishSnapshot)

	mux.HandleFunc("GET /api/selfcheck", h.selfCheck)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// httpStatus 将领域错误映射为 HTTP 状态码。
func httpStatus(err error) int {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrAlreadyExists),
		errors.Is(err, model.ErrDatasetSealed),
		errors.Is(err, model.ErrSnapshotSealed),
		errors.Is(err, model.ErrReleaseConflict),
		errors.Is(err, model.ErrRevokeMissing):
		return http.StatusConflict
	case errors.Is(err, model.ErrDatasetCycle),
		errors.Is(err, model.ErrParentMissing),
		errors.Is(err, model.ErrMechanismInvalid),
		errors.Is(err, model.ErrIllegalEpsilonDelta),
		errors.Is(err, model.ErrUnknownRule),
		errors.Is(err, model.ErrUnknownKind),
		errors.Is(err, model.ErrDatasetMissing),
		errors.Is(err, model.ErrMechanismMissing):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, httpStatus(err), map[string]string{"error": err.Error()})
}
