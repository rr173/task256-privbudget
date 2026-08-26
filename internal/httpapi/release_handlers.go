package httpapi

import (
	"net/http"

	"task256-privbudget/internal/model"
)

// createRelease 创建发布批次。
func (h *Handler) createRelease(w http.ResponseWriter, r *http.Request) {
	var rel model.Release
	if err := readJSON(r, &rel); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	ctx := r.Context()
	if err := h.app.CreateRelease(ctx, rel); err != nil {
		writeErr(w, err)
		return
	}
	stored, err := h.app.GetRelease(ctx, rel.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}

// listReleases 列出全部发布。
func (h *Handler) listReleases(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.ListReleases(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// getRelease 取单个发布。
func (h *Handler) getRelease(w http.ResponseWriter, r *http.Request) {
	rel, err := h.app.GetRelease(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

// evaluateRelease 评估发布（允许/拒绝）。
func (h *Handler) evaluateRelease(w http.ResponseWriter, r *http.Request) {
	rep, err := h.app.EvaluateRelease(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"release": r.PathValue("id"),
		"report":  rep,
	})
}

// revokeRelease 撤销已允许的发布。
func (h *Handler) revokeRelease(w http.ResponseWriter, r *http.Request) {
	rep, err := h.app.RevokeRelease(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"revoked": r.PathValue("id"),
		"report":  rep,
	})
}
