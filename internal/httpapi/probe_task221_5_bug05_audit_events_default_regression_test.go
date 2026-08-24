package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task221-packthermal/internal/service"
	"task221-packthermal/internal/store"
)

func TestBug05AuditEventsDefaultLimitReturnsEvents(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := service.New(store.NewRepositories(db))
	if err := svc.RunDemo(t.Context()); err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()
	New(svc).Handler().ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/audit/events", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	var body struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Count == 0 {
		t.Fatal("default audit listing returned no events")
	}
}
