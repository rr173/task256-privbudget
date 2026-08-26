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

func task256Handler3(t *testing.T) http.Handler { st, err := store.Open(filepath.Join(t.TempDir(), "probe.db")); if err != nil { t.Fatal(err) }; t.Cleanup(func(){ _ = st.Close() }); return httpapi.NewHandler(service.NewApp(st)).Routes() }
func task256Call3(t *testing.T, h http.Handler, method, path string, v any) (int, []byte) { b, _ := json.Marshal(v); req := httptest.NewRequest(method, path, bytes.NewReader(b)); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req); return rec.Code, rec.Body.Bytes() }

func TestBug03_SealedDatasetRejectsNewRelease(t *testing.T) {
	h := task256Handler3(t)
	steps := []struct{ method, path string; body any; want int }{
		{http.MethodPost, "/api/datasets", model.DatasetVersion{ID:"ds", Name:"population", Version:"v1", EpsilonCap:10, DeltaCap:1}, http.StatusCreated},
		{http.MethodPost, "/api/mechanisms", model.Mechanism{ID:"m", Kind:model.MechLaplace, Epsilon:.2, DatasetIDs:[]string{"ds"}}, http.StatusCreated},
		{http.MethodPost, "/api/mechanisms/m/verify", map[string]any{}, http.StatusOK},
		{http.MethodPost, "/api/datasets/ds/seal", map[string]any{}, http.StatusOK},
		{http.MethodPost, "/api/releases", model.Release{ID:"r", MechanismID:"m", Rule:model.RuleSequential}, http.StatusCreated},
	}
	for _, s := range steps { if code, body := task256Call3(t, h, s.method, s.path, s.body); code != s.want { t.Fatalf("%s %s status=%d want=%d body=%s", s.method, s.path, code, s.want, body) } }
	code, body := task256Call3(t, h, http.MethodPost, "/api/releases/r/evaluate", map[string]any{})
	if code != http.StatusConflict { t.Fatalf("sealed dataset release evaluated: status=%d body=%s", code, body) }
	code, body = task256Call3(t, h, http.MethodGet, "/api/releases/r", map[string]any{})
	if code != http.StatusOK { t.Fatalf("get release status=%d", code) }
	var got model.Release
	if err := json.Unmarshal(body, &got); err != nil { t.Fatal(err) }
	if got.Status != model.ReleasePending { t.Fatalf("failed evaluation changed release to %q", got.Status) }
}
