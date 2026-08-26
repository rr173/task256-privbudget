package httpapi_test

import (
	"bytes"
	"database/sql"
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

func task256Handler9(t *testing.T) (http.Handler, *sql.DB) { st, err := store.Open(filepath.Join(t.TempDir(), "probe.db")); if err != nil { t.Fatal(err) }; t.Cleanup(func(){ _ = st.Close() }); return httpapi.NewHandler(service.NewApp(st)).Routes(), st.DB() }
func task256Call9(t *testing.T, h http.Handler, method, path string, v any) (int, []byte) { b, _ := json.Marshal(v); req := httptest.NewRequest(method, path, bytes.NewReader(b)); rec := httptest.NewRecorder(); h.ServeHTTP(rec, req); return rec.Code, rec.Body.Bytes() }

func TestBug09_SnapshotPublishFailureIsAtomic(t *testing.T) {
	h, db := task256Handler9(t)
	for _, id := range []string{"old", "new"} {
		if code, body := task256Call9(t, h, http.MethodPost, "/api/snapshots", model.BudgetSnapshot{ID:id, Name:id, Rule:model.RuleSequential, Status:model.SnapDraft}); code != http.StatusCreated { t.Fatalf("create %s status=%d body=%s", id, code, body) }
	}
	if code, body := task256Call9(t, h, http.MethodPost, "/api/snapshots/old/publish", map[string]any{}); code != http.StatusOK { t.Fatalf("publish old status=%d body=%s", code, body) }
	const trigger = `CREATE TRIGGER probe_fail_new_publish BEFORE UPDATE OF status ON snapshots WHEN NEW.id='new' AND NEW.status='published' BEGIN SELECT RAISE(ABORT, 'probe publish failure'); END`
	if _, err := db.Exec(trigger); err != nil { t.Fatal(err) }
	if code, _ := task256Call9(t, h, http.MethodPost, "/api/snapshots/new/publish", map[string]any{}); code == http.StatusOK { t.Fatal("forced publish failure unexpectedly succeeded") }
	code, body := task256Call9(t, h, http.MethodGet, "/api/snapshots/old", map[string]any{})
	if code != http.StatusOK { t.Fatalf("get old status=%d", code) }
	var old model.BudgetSnapshot
	if err := json.Unmarshal(body, &old); err != nil { t.Fatal(err) }
	if old.Status != model.SnapPublished { t.Fatalf("old snapshot became %q after failed publish", old.Status) }
	code, body = task256Call9(t, h, http.MethodGet, "/api/snapshots/new", map[string]any{})
	if code != http.StatusOK { t.Fatalf("get new status=%d", code) }
	var next model.BudgetSnapshot
	if err := json.Unmarshal(body, &next); err != nil { t.Fatal(err) }
	if next.Status != model.SnapDraft { t.Fatalf("new snapshot became %q after failed publish", next.Status) }
}
