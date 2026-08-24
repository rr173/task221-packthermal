package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"task221-packthermal/internal/service"
	"task221-packthermal/internal/store"
)

func TestBug06CancelledSnapshotDoesNotPersist(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "cancel-snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := service.New(store.NewRepositories(db))
	if err := svc.RunDemo(t.Context()); err != nil {
		t.Fatal(err)
	}
	trials, _ := svc.ListTrials(t.Context())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/trials/"+strconv.FormatInt(trials[0].ID, 10)+"/snapshots", nil).WithContext(ctx)
	resp := httptest.NewRecorder()
	New(svc).Handler().ServeHTTP(resp, req)
	if resp.Code == http.StatusCreated {
		t.Fatalf("cancelled publish succeeded: %s", resp.Body.String())
	}
	snaps, err := svc.ListSnapshots(t.Context(), trials[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 {
		b, _ := json.Marshal(snaps)
		t.Fatalf("cancelled publish persisted snapshots=%s", b)
	}
}
