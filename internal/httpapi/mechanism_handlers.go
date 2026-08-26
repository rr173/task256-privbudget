package httpapi

import (
	"net/http"

	"task256-privbudget/internal/model"
)

// registerMechanism 登记统计机制。
func (h *Handler) registerMechanism(w http.ResponseWriter, r *http.Request) {
	var m model.Mechanism
	if err := readJSON(r, &m); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	ctx := r.Context()
	if err := h.app.RegisterMechanism(ctx, m); err != nil {
		writeErr(w, err)
		return
	}
	stored, err := h.app.GetMechanism(ctx, m.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}

// listMechanisms 列出全部机制。
func (h *Handler) listMechanisms(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.ListMechanisms(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// getMechanism 取单个机制。
func (h *Handler) getMechanism(w http.ResponseWriter, r *http.Request) {
	m, err := h.app.GetMechanism(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// verifyMechanism 验证机制。
func (h *Handler) verifyMechanism(w http.ResponseWriter, r *http.Request) {
	if err := h.app.VerifyMechanism(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

// revokeMechanism 撤销机制。
func (h *Handler) revokeMechanism(w http.ResponseWriter, r *http.Request) {
	if err := h.app.RevokeMechanism(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
