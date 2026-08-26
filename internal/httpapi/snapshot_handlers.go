package httpapi

import (
	"net/http"

	"task256-privbudget/internal/model"
)

// createSnapshot 创建草稿快照。
func (h *Handler) createSnapshot(w http.ResponseWriter, r *http.Request) {
	var snap model.BudgetSnapshot
	if err := readJSON(r, &snap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	ctx := r.Context()
	if err := h.app.CreateSnapshot(ctx, snap); err != nil {
		writeErr(w, err)
		return
	}
	stored, err := h.app.GetSnapshot(ctx, snap.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}

// listSnapshots 列出全部快照。
func (h *Handler) listSnapshots(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.ListSnapshots(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// getSnapshot 取单个快照。
func (h *Handler) getSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := h.app.GetSnapshot(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// publishSnapshot 发布（冻结）草稿快照。
func (h *Handler) publishSnapshot(w http.ResponseWriter, r *http.Request) {
	rule, err := parseRule(r, nil)
	if err != nil {
		writeErr(w, err)
		return
	}
	rep, err := h.app.PublishSnapshot(r.Context(), r.PathValue("id"), rule)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"snapshot": r.PathValue("id"),
		"report":   rep,
	})
}
