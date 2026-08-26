package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task256-privbudget/internal/httpapi"
	"task256-privbudget/internal/model"
	"task256-privbudget/internal/service"
	"task256-privbudget/internal/store"
)

func newTask256Handler(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = st.Close() })
	return httpapi.NewHandler(service.NewApp(st)).Routes()
}

func task256JSON(t *testing.T, h http.Handler, method, path string, body any) (int, []byte) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil { t.Fatal(err) }
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func TestBug01_ClientCannotPreAllowRelease(t *testing.T) {
	h := newTask256Handler(t)
	if code, _ := task256JSON(t, h, http.MethodPost, "/api/datasets", model.DatasetVersion{ID: "ds", Name: "population", Version: "v1", EpsilonCap: 10, DeltaCap: 1}); code != http.StatusCreated { t.Fatalf("dataset status=%d", code) }
	if code, _ := task256JSON(t, h, http.MethodPost, "/api/mechanisms", model.Mechanism{ID: "m", Kind: model.MechLaplace, Epsilon: .6, DatasetIDs: []string{"ds"}}); code != http.StatusCreated { t.Fatalf("mechanism status=%d", code) }
	if code, _ := task256JSON(t, h, http.MethodPost, "/api/mechanisms/m/verify", map[string]any{}); code != http.StatusOK { t.Fatalf("verify status=%d", code) }
	code, body := task256JSON(t, h, http.MethodPost, "/api/releases", model.Release{ID: "r", MechanismID: "m", Rule: model.RuleSequential, Status: model.ReleaseAllowed})
	if code != http.StatusCreated { t.Fatalf("create release status=%d body=%s", code, body) }
	var got model.Release
	if err := json.Unmarshal(body, &got); err != nil { t.Fatal(err) }
	if got.Status != model.ReleasePending { t.Fatalf("release status=%q, want pending", got.Status) }
	code, body = task256JSON(t, h, http.MethodGet, "/api/budget?rule=sequential", map[string]any{})
	if code != http.StatusOK { t.Fatalf("budget status=%d body=%s", code, body) }
	var rep struct{ Entries []model.BudgetEntry `json:"entries"` }
	if err := json.Unmarshal(body, &rep); err != nil { t.Fatal(err) }
	if len(rep.Entries) != 0 { t.Fatalf("unassessed release consumed budget: %+v", rep.Entries) }
}
