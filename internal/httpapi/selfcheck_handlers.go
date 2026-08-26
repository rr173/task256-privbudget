package httpapi

import "net/http"

// selfCheck 运行内部不变量自检。
func (h *Handler) selfCheck(w http.ResponseWriter, r *http.Request) {
	issues, err := h.app.SelfCheck(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"ok":     len(issues) == 1 && issues[0] == "ok",
	})
}
