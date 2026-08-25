package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/service"
	"task221-packthermal/internal/store"
)

func TestBug02DuplicateSeriesUsesConflictResponse(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "duplicate.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := service.New(store.NewRepositories(db))
	trial, err := svc.CreateTrial(t.Context(), "TRL-DUP", "duplicate")
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	sensor, err := svc.AddSensor(t.Context(), trial.ID, "S1", "inner", "celsius")
	if err != nil {
		t.Fatalf("add sensor: %v", err)
	}
	handler := New(svc).Handler()
	body := map[string]any{"sensor_id": sensor.ID, "scale": "celsius", "samples": []model.TempSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 10}}}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	post := func() *httptest.ResponseRecorder {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/trials/"+strconv.FormatInt(trial.ID, 10)+"/tempseries", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(resp, req)
		return resp
	}
	first := post()
	if first.Code != http.StatusCreated {
		t.Fatalf("first upload status = %d, body=%s", first.Code, first.Body.String())
	}
	second := post()
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate upload status = %d, want %d; body=%s", second.Code, http.StatusConflict, second.Body.String())
	}
}
