package httpapi_test

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task256-privbudget/internal/compose"
	"task256-privbudget/internal/httpapi"
	"task256-privbudget/internal/model"
	"task256-privbudget/internal/service"
	"task256-privbudget/internal/store"
)

func task256Handler10(t *testing.T) http.Handler { st, err := store.Open(filepath.Join(t.TempDir(), "probe.db")); if err != nil { t.Fatal(err) }; t.Cleanup(func(){ _ = st.Close() }); return httpapi.NewHandler(service.NewApp(st)).Routes() }
func task256Call10(t *testing.T, h http.Handler, method, path string, v any) (int, []byte) { b, _ := json.Marshal(v); req := httptest.NewRequest(method, path, bytes.NewReader(b)); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req); return rec.Code, rec.Body.Bytes() }

func TestBug10_AdvancedBudgetIncludesNonlinearCorrection(t *testing.T) {
	h := task256Handler10(t)
	steps := []struct{ method, path string; body any; want int }{
		{http.MethodPost, "/api/datasets", model.DatasetVersion{ID:"ds", Name:"population", Version:"v1", EpsilonCap:100, DeltaCap:1}, http.StatusCreated},
		{http.MethodPost, "/api/mechanisms", model.Mechanism{ID:"m1", Kind:model.MechLaplace, Epsilon:.05, DatasetIDs:[]string{"ds"}}, http.StatusCreated},
		{http.MethodPost, "/api/mechanisms", model.Mechanism{ID:"m2", Kind:model.MechLaplace, Epsilon:.05, DatasetIDs:[]string{"ds"}}, http.StatusCreated},
		{http.MethodPost, "/api/mechanisms/m1/verify", map[string]any{}, http.StatusOK},
		{http.MethodPost, "/api/mechanisms/m2/verify", map[string]any{}, http.StatusOK},
		{http.MethodPost, "/api/releases", model.Release{ID:"r1", MechanismID:"m1", Rule:model.RuleSequential}, http.StatusCreated},
		{http.MethodPost, "/api/releases", model.Release{ID:"r2", MechanismID:"m2", Rule:model.RuleSequential}, http.StatusCreated},
		{http.MethodPost, "/api/releases/r1/evaluate", map[string]any{}, http.StatusOK},
		{http.MethodPost, "/api/releases/r2/evaluate", map[string]any{}, http.StatusOK},
	}
	for _, s := range steps { if code, body := task256Call10(t, h, s.method, s.path, s.body); code != s.want { t.Fatalf("%s %s status=%d want=%d body=%s", s.method, s.path, code, s.want, body) } }
	code, body := task256Call10(t, h, http.MethodGet, "/api/budget?rule=advanced", map[string]any{})
	if code != http.StatusOK { t.Fatalf("advanced status=%d body=%s", code, body) }
	var rep compose.Report
	if err := json.Unmarshal(body, &rep); err != nil { t.Fatal(err) }
	if len(rep.Entries) != 1 { t.Fatalf("entries=%+v", rep.Entries) }
	want := math.Sqrt(2*math.Log(1e6)*(2*.05*.05)) + 2*.05*(math.Exp(.05)-1)
	got := rep.Entries[0].EpsilonUsed
	if math.Abs(got-want) > 1e-6 { t.Fatalf("advanced epsilon=%0.9f want safe bound %0.9f", got, want) }
}
