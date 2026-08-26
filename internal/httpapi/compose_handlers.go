package httpapi

import (
	"net/http"

	"task256-privbudget/internal/model"
)

// parseRule 从查询参数或请求体解析组合规则，缺省为顺序组合。
func parseRule(r *http.Request, body map[string]interface{}) model.CompositionRule {
	rule := model.RuleSequential
	if v := r.URL.Query().Get("rule"); v != "" {
		rule = model.CompositionRule(v)
	} else if body != nil {
		if rv, ok := body["rule"].(string); ok && rv != "" {
			rule = model.CompositionRule(rv)
		}
	}
	switch rule {
	case model.RuleSequential, model.RuleAdvanced, model.RuleParallel, model.RuleRDP:
		return rule
	default:
		return model.RuleSequential
	}
}

// evaluateBudget GET 评估当前预算（?rule=）。
func (h *Handler) evaluateBudget(w http.ResponseWriter, r *http.Request) {
	rule := parseRule(r, nil)
	rep, err := h.app.EvaluateBudget(r.Context(), rule)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// evaluateBudgetPost POST 评估当前预算（支持请求体指定规则）。
func (h *Handler) evaluateBudgetPost(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	_ = readJSON(r, &body)
	rule := parseRule(r, body)
	rep, err := h.app.EvaluateBudget(r.Context(), rule)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
