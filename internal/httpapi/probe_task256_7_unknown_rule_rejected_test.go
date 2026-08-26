package httpapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task256-privbudget/internal/httpapi"
	"task256-privbudget/internal/service"
	"task256-privbudget/internal/store"
)

func task256Handler7(t *testing.T) http.Handler { st, err := store.Open(filepath.Join(t.TempDir(), "probe.db")); if err != nil { t.Fatal(err) }; t.Cleanup(func(){ _ = st.Close() }); return httpapi.NewHandler(service.NewApp(st)).Routes() }

func TestBug07_UnknownBudgetRuleIsRejected(t *testing.T) {
	h := task256Handler7(t)
	req := httptest.NewRequest(http.MethodGet, "/api/budget?rule=not-a-rule", bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest { t.Fatalf("unknown rule silently accepted: status=%d body=%s", rec.Code, rec.Body.String()) }
}
