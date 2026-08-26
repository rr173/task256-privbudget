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

func task256Handler8(t *testing.T) http.Handler { st, err := store.Open(filepath.Join(t.TempDir(), "probe.db")); if err != nil { t.Fatal(err) }; t.Cleanup(func(){ _ = st.Close() }); return httpapi.NewHandler(service.NewApp(st)).Routes() }
func task256Call8(t *testing.T, h http.Handler, method, path string, v any) (int, []byte) { b, _ := json.Marshal(v); req := httptest.NewRequest(method, path, bytes.NewReader(b)); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req); return rec.Code, rec.Body.Bytes() }

func TestBug08_DraftSnapshotHasNoFrozenTime(t *testing.T) {
	h := task256Handler8(t)
	if code, body := task256Call8(t, h, http.MethodPost, "/api/snapshots", model.BudgetSnapshot{ID:"s1", Name:"draft", Rule:model.RuleSequential, Status:model.SnapDraft}); code != http.StatusCreated { t.Fatalf("create status=%d body=%s", code, body) }
	code, body := task256Call8(t, h, http.MethodGet, "/api/snapshots/s1", map[string]any{})
	if code != http.StatusOK { t.Fatalf("get status=%d", code) }
	var got model.BudgetSnapshot
	if err := json.Unmarshal(body, &got); err != nil { t.Fatal(err) }
	if got.Status != model.SnapDraft { t.Fatalf("status=%q want draft", got.Status) }
	if !got.FrozenAt.IsZero() { t.Fatalf("draft snapshot already frozen at %s", got.FrozenAt.Format("2006-01-02T15:04:05.999999999Z07:00")) }
}
