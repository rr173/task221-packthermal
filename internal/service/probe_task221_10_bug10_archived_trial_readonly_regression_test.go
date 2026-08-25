package service

import (
	"context"
	"path/filepath"
	"testing"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/store"
)

func TestBug10ArchivedTrialRejectsLayerWrite(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "archived.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := New(store.NewRepositories(db))
	ctx := context.Background()
	tr, err := svc.CreateTrial(ctx, "TRL-ARCHIVE", "archived")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct{ from, to string }{{model.TrialPlanned, model.TrialCollecting}, {model.TrialCollecting, model.TrialPendingInversion}, {model.TrialPendingInversion, model.TrialConfirmed}, {model.TrialConfirmed, model.TrialArchived}} {
		if err := svc.repos.Trial.TransitionState(ctx, tr.ID, step.from, step.to); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.AddLayer(ctx, tr.ID, &model.Layer{Seq: 1, Material: "wall", ThicknessMM: 1, Conductivity: 1, Density: 1, SpecificHeat: 1, AreaM2: 1}); err != model.ErrImmutable {
		t.Fatalf("archived write error=%v, want ErrImmutable", err)
	}
}
