package httpapi

import (
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

func TestBug04RepublishLeavesNewestSnapshotPublished(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := service.New(store.NewRepositories(db))
	if err := svc.RunDemo(t.Context()); err != nil {
		t.Fatal(err)
	}
	trials, err := svc.ListTrials(t.Context())
	if err != nil || len(trials) != 1 {
		t.Fatalf("trials=%+v err=%v", trials, err)
	}
	resp := httptest.NewRecorder()
	New(svc).Handler().ServeHTTP(resp, httptest.NewRequest(http.MethodPost, "/api/trials/"+strconv.FormatInt(trials[0].ID, 10)+"/snapshots", nil))
	if resp.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	var latest model.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		t.Fatal(err)
	}
	if latest.State != model.SnapshotPublished {
		t.Fatalf("response state=%s", latest.State)
	}
	snaps, err := svc.ListSnapshots(t.Context(), trials[0].ID)
	if err != nil || len(snaps) != 2 {
		t.Fatalf("snapshots=%+v err=%v", snaps, err)
	}
	if snaps[0].State != model.SnapshotSuperseded || snaps[1].State != model.SnapshotPublished {
		t.Fatalf("states=%s,%s", snaps[0].State, snaps[1].State)
	}
}
