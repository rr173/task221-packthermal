package store

import (
	"context"
	"path/filepath"
	"testing"

	"task221-packthermal/internal/model"
)

func TestTrialPersistsAcrossCloseAndReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "recovery.db")
	ctx := context.Background()

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open initial store: %v", err)
	}
	repos := NewRepositories(db)
	created, err := repos.Trial.CreateTrial(ctx, "TRL-RECOVER", "restart recovery")
	if err != nil {
		db.Close()
		t.Fatalf("create trial: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close initial store: %v", err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	restored, err := NewRepositories(reopened).Trial.GetTrial(ctx, created.ID)
	if err != nil {
		t.Fatalf("restore trial: %v", err)
	}
	if restored.Code != created.Code || restored.Title != created.Title || restored.State != model.TrialPlanned {
		t.Fatalf("restored trial = %+v, want original planned trial", restored)
	}
}
