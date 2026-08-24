package service

import (
	"context"
	"path/filepath"
	"testing"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/store"
)

func TestBug07MissingReferenceReturnsDomainError(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "missing-reference.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := New(store.NewRepositories(db))
	ctx := context.Background()
	trial, err := svc.CreateTrial(ctx, "TRL-NOREF", "missing reference")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AddLayer(ctx, trial.ID, &model.Layer{Seq: 1, Material: "wall", ThicknessMM: 2, Conductivity: 0.4, Density: 900, SpecificHeat: 1800, AreaM2: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AddEnv(ctx, trial.ID, "step", []model.EnvSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 25}}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.StartCollecting(ctx, trial.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.FreezeTrial(ctx, trial.ID); err != nil {
		t.Fatal(err)
	}
	m, err := svc.CreateModel(ctx, trial.ID, "model")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ConfirmModel(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.RunInversion(ctx, trial.ID, m.ID); err == nil {
		t.Fatal("missing reference should return an error")
	}
}
