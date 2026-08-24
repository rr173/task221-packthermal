package service

import (
	"context"
	"path/filepath"
	"testing"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/store"
)

func TestBug08EmptyNetworkReturnsDomainError(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "empty-network.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := New(store.NewRepositories(db))
	ctx := context.Background()
	trial, err := svc.CreateTrial(ctx, "TRL-EMPTY", "empty network")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AddEnv(ctx, trial.ID, "step", []model.EnvSample{{Seconds: 0, Temperature: 5}, {Seconds: 60, Temperature: 25}}); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.AddSensor(ctx, trial.ID, "S1", "inner", "celsius"); err != nil {
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
		t.Fatal("empty network should return an error")
	}
}
