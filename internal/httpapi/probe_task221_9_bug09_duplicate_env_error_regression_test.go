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

func TestBug09DuplicateEnvUsesConflictResponse(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "duplicate-env.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := service.New(store.NewRepositories(db))
	trial, err := svc.CreateTrial(t.Context(), "TRL-ENV-DUP", "duplicate env")
	if err != nil {
		t.Fatal(err)
	}
	h := New(svc).Handler()
	body, _ := json.Marshal(map[string]any{"name": "step", "samples": []model.EnvSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 25}}})
	post := func() *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/trials/"+strconv.FormatInt(trial.ID, 10)+"/env", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(r, req)
		return r
	}
	if first := post(); first.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	if second := post(); second.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", second.Code, second.Body.String())
	}
}
