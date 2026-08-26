package httpapi

import (
	"net/http"

	"task256-privbudget/internal/model"
)

// parseRule 从查询参数或请求体解析组合规则，缺省为顺序组合。
// 当显式指定了不支持的规则时返回 ErrUnknownRule，调用方据此返回参数错误，
// 绝不静默改用其它规则并返回看似成功的预算报告。
func parseRule(r *http.Request, body map[string]interface{}) (model.CompositionRule, error) {
	rule := model.RuleSequential
	provided := false
	if v := r.URL.Query().Get("rule"); v != "" {
		rule = model.CompositionRule(v)
		provided = true
	} else if body != nil {
		if rv, ok := body["rule"].(string); ok && rv != "" {
			rule = model.CompositionRule(rv)
			provided = true
		}
	}
	if !provided {
		return model.RuleSequential, nil
	}
	if !model.IsSupportedRule(rule) {
		return "", model.ErrUnknownRule
	}
	return rule, nil
}

// evaluateBudget GET 评估当前预算（?rule=）。
func (h *Handler) evaluateBudget(w http.ResponseWriter, r *http.Request) {
	rule, err := parseRule(r, nil)
	if err != nil {
		writeErr(w, err)
		return
	}
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
	rule, err := parseRule(r, body)
	if err != nil {
		writeErr(w, err)
		return
	}
	rep, err := h.app.EvaluateBudget(r.Context(), rule)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
