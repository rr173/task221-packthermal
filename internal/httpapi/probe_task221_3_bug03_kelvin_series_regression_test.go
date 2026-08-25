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

func TestBug03KelvinSeriesIsStoredInCelsius(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "kelvin.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := service.New(store.NewRepositories(db))
	trial, err := svc.CreateTrial(t.Context(), "TRL-KELVIN", "kelvin")
	if err != nil {
		t.Fatal(err)
	}
	sensor, err := svc.AddSensor(t.Context(), trial.ID, "S1", "inner", "kelvin")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"sensor_id": sensor.ID, "scale": "kelvin", "samples": []model.TempSample{{Seconds: 0, Temperature: 278.15}, {Seconds: 60, Temperature: 283.15}}})
	req := httptest.NewRequest(http.MethodPost, "/api/trials/"+strconv.FormatInt(trial.ID, 10)+"/tempseries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	New(svc).Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	series, err := svc.ListSeries(t.Context(), trial.ID)
	if err != nil || len(series) != 1 {
		t.Fatalf("series=%+v err=%v", series, err)
	}
	samples, err := store.NewRepositories(db).Series.LoadSamples(t.Context(), series[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if samples[0].Temperature != 5 || samples[1].Temperature != 10 {
		t.Fatalf("stored samples=%+v", samples)
	}
}
