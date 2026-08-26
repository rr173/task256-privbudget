package httpapi

import (
	"net/http"

	"task256-privbudget/internal/model"
)

// registerDataset 登记数据集版本。
func (h *Handler) registerDataset(w http.ResponseWriter, r *http.Request) {
	var d model.DatasetVersion
	if err := readJSON(r, &d); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	ctx := r.Context()
	if err := h.app.RegisterDataset(ctx, d); err != nil {
		writeErr(w, err)
		return
	}
	stored, err := h.app.GetDataset(ctx, d.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}

// listDatasets 列出全部数据集版本。
func (h *Handler) listDatasets(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.ListDatasets(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// getDataset 取单个数据集版本。
func (h *Handler) getDataset(w http.ResponseWriter, r *http.Request) {
	d, err := h.app.GetDataset(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// updateDataset 更新数据集版本。
func (h *Handler) updateDataset(w http.ResponseWriter, r *http.Request) {
	var d model.DatasetVersion
	if err := readJSON(r, &d); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	d.ID = r.PathValue("id")
	if err := h.app.UpdateDataset(r.Context(), d); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// sealDataset 封存数据集版本。
func (h *Handler) sealDataset(w http.ResponseWriter, r *http.Request) {
	if err := h.app.SealDataset(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sealed"})
}
