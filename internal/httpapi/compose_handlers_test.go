package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/service"
	"task256-privbudget/internal/store"
)

// newTestHandler 建立一个最小可用的应用并返回其 HTTP 处理器，
// 用于对预算规则解析做端到端断言。
func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	app := service.NewApp(st)

	if err := app.RegisterDataset(ctx, model.DatasetVersion{
		ID: "ds", Name: "pop", Version: "v1",
		Status: model.DatasetRegistered, EpsilonCap: 1.0, DeltaCap: 1e-5,
	}); err != nil {
		t.Fatal(err)
	}
	m := model.Mechanism{ID: "m", Kind: model.MechLaplace, Epsilon: 0.3, Delta: 0, DatasetIDs: []string{"ds"}, Status: model.MechDraft}
	if err := app.RegisterMechanism(ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyMechanism(ctx, "m"); err != nil {
		t.Fatal(err)
	}
	return NewHandler(app)
}

// do 规范化地发起一个请求并返回响应与解码后的错误体。
func do(t *testing.T, h http.Handler, method, target, body string, hdr map[string]string) (*http.Response, map[string]string) {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	errBody := map[string]string{}
	_ = json.NewDecoder(rec.Body).Decode(&errBody)
	return rec.Result(), errBody
}

// TestBudgetRejectsUnsupportedRule 断言：当请求一个服务不支持的组合规则时，
// 各预算端点必须以参数错误（400）拒绝，绝不静默改用其它规则并返回看似成功的预算报告。
func TestBudgetRejectsUnsupportedRule(t *testing.T) {
	h := newTestHandler(t)

	for _, tc := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{"GET budget", "GET", "/api/budget?rule=graph", ""},
		{"GET budget typo", "GET", "/api/budget?rule=seqwential", ""},
		{"GET budget unknown", "GET", "/api/budget?rule=zeta", ""},
		{"POST budget evaluate body", "POST", "/api/budget/evaluate", `{"rule":"graph"}`},
		{"POST budget evaluate query", "POST", "/api/budget/evaluate?rule=graph", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, errBody := do(t, h.Routes(), tc.method, tc.target, tc.body, nil)
			resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("unsupported rule %q: expected 400, got %d (body=%v)", tc.target, resp.StatusCode, errBody)
			}
			if !strings.Contains(errBody["error"], model.ErrUnknownRule.Error()) {
				t.Fatalf("expected error mentioning %q, got %q", model.ErrUnknownRule.Error(), errBody["error"])
			}
		})
	}
}

// TestBudgetSupportedRulesStillWork 断言：合法规则仍返回 200 与预算报告，
// 确保拒绝逻辑没有误伤受支持的规则。
func TestBudgetSupportedRulesStillWork(t *testing.T) {
	h := newTestHandler(t)
	for _, rule := range model.SupportedCompositionRules() {
		t.Run(string(rule), func(t *testing.T) {
			resp, errBody := do(t, h.Routes(), "GET", "/api/budget?rule="+string(rule), "", nil)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("supported rule %s: expected 200, got %d (body=%v)", rule, resp.StatusCode, errBody)
			}
		})
	}
}

// TestPublishSnapshotRejectsUnsupportedRule 断言：用不支持的规则发布快照时
// 必须拒绝（400），且快照不得被冻结为已发布。
func TestPublishSnapshotRejectsUnsupportedRule(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t)
	snap := model.BudgetSnapshot{ID: "s1", Name: "snap", Rule: model.RuleSequential, Status: model.SnapDraft}
	if err := h.app.CreateSnapshot(ctx, snap); err != nil {
		t.Fatal(err)
	}

	resp, errBody := do(t, h.Routes(), "POST", "/api/snapshots/s1/publish?rule=graph", "", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unsupported publish rule: expected 400, got %d (body=%v)", resp.StatusCode, errBody)
	}
	if !strings.Contains(errBody["error"], model.ErrUnknownRule.Error()) {
		t.Fatalf("expected error mentioning %q, got %q", model.ErrUnknownRule.Error(), errBody["error"])
	}

	got, err := h.app.GetSnapshot(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.SnapDraft {
		t.Fatalf("snapshot must remain draft after rejected publish, got %s", got.Status)
	}
}
