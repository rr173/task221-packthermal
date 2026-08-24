package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task221-packthermal/internal/service"
	"task221-packthermal/internal/store"
)

func TestBug01CancelledCreateDoesNotPersist(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	handler := New(service.New(store.NewRepositories(db))).Handler()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/trials", strings.NewReader(`{"code":"TRL-CANCEL","title":"cancelled"}`)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code == http.StatusCreated {
		t.Fatalf("cancelled request unexpectedly created a trial: %s", resp.Body.String())
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/trials", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d", list.Code)
	}
	var body struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(list.Body).Decode(&body); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if body.Count != 0 {
		t.Fatalf("cancelled request persisted %d trial(s)", body.Count)
	}
}
