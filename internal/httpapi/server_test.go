package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/service"
	"task221-packthermal/internal/store"
)

func TestHandlerWiresHealthAndTrialRoutes(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	handler := New(service.New(store.NewRepositories(db))).Handler()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", health.Code, http.StatusOK)
	}

	createBody := bytes.NewBufferString(`{"code":"TRL-HTTP","title":"route test"}`)
	create := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/trials", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(create, createReq)
	if create.Code != http.StatusCreated {
		t.Fatalf("POST /api/trials status = %d, want %d; body=%s", create.Code, http.StatusCreated, create.Body.String())
	}
	var trial model.Trial
	if err := json.NewDecoder(create.Body).Decode(&trial); err != nil {
		t.Fatalf("decode created trial: %v", err)
	}
	if trial.Code != "TRL-HTTP" || trial.State != model.TrialPlanned {
		t.Fatalf("created trial = %+v, want code/state TRL-HTTP/planned", trial)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/trials", nil).WithContext(context.Background()))
	if list.Code != http.StatusOK {
		t.Fatalf("GET /api/trials status = %d, want %d", list.Code, http.StatusOK)
	}
	var payload struct {
		Trials []model.Trial `json:"trials"`
		Count  int           `json:"count"`
	}
	if err := json.NewDecoder(list.Body).Decode(&payload); err != nil {
		t.Fatalf("decode trial list: %v", err)
	}
	if payload.Count != 1 || len(payload.Trials) != 1 || payload.Trials[0].ID != trial.ID {
		t.Fatalf("trial list = %+v, want one created trial", payload)
	}
}
