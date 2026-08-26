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

func task256Handler2(t *testing.T) http.Handler { st, err := store.Open(filepath.Join(t.TempDir(), "probe.db")); if err != nil { t.Fatal(err) }; t.Cleanup(func(){ _ = st.Close() }); return httpapi.NewHandler(service.NewApp(st)).Routes() }
func task256Call2(t *testing.T, h http.Handler, method, path string, v any) (int, []byte) { b, _ := json.Marshal(v); req := httptest.NewRequest(method, path, bytes.NewReader(b)); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req); return rec.Code, rec.Body.Bytes() }

func TestBug02_SealedDatasetRejectsNewMechanism(t *testing.T) {
	h := task256Handler2(t)
	if code, _ := task256Call2(t, h, http.MethodPost, "/api/datasets", model.DatasetVersion{ID:"ds", Name:"population", Version:"v1", EpsilonCap:1, DeltaCap:1}); code != http.StatusCreated { t.Fatalf("dataset status=%d", code) }
	if code, _ := task256Call2(t, h, http.MethodPost, "/api/datasets/ds/seal", map[string]any{}); code != http.StatusOK { t.Fatalf("seal status=%d", code) }
	code, body := task256Call2(t, h, http.MethodPost, "/api/mechanisms", model.Mechanism{ID:"m", Kind:model.MechLaplace, Epsilon:.2, DatasetIDs:[]string{"ds"}})
	if code != http.StatusConflict { t.Fatalf("sealed dataset accepted mechanism: status=%d body=%s", code, body) }
	code, body = task256Call2(t, h, http.MethodGet, "/api/mechanisms", map[string]any{})
	if code != http.StatusOK { t.Fatalf("list status=%d", code) }
	var list []model.Mechanism
	if err := json.Unmarshal(body, &list); err != nil { t.Fatal(err) }
	if len(list) != 0 { t.Fatalf("rejected mechanism persisted: %+v", list) }
}
