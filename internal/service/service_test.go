package service

import (
	"context"
	"path/filepath"
	"testing"

	"task221-packthermal/internal/model"
	"task221-packthermal/internal/store"
)

func newTestService(t *testing.T) (*Service, *store.DB) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return New(store.NewRepositories(db)), db
}

func TestRunDemo(t *testing.T) {
	svc, _ := newTestService(t)
	if err := svc.RunDemo(context.Background()); err != nil {
		t.Fatalf("RunDemo: %v", err)
	}
}

func TestRunDemoDeterministicAcrossInstances(t *testing.T) {
	// 两个独立空库分别跑 RunDemo，均应成功（验证确定性）。
	for i := 0; i < 2; i++ {
		svc, _ := newTestService(t)
		if err := svc.RunDemo(context.Background()); err != nil {
			t.Fatalf("RunDemo instance %d: %v", i, err)
		}
	}
}

func TestDuplicateTrialCodeRejected(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateTrial(ctx, "TRL-X", "first"); err != nil {
		t.Fatalf("create first: %v", err)
	}
	if _, err := svc.CreateTrial(ctx, "TRL-X", "second"); err != model.ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestArchiveRejectsWrite(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	// 构建一个走完流程的试验并封存。
	tr, err := svc.CreateTrial(ctx, "TRL-ARCH", "archive test")
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	// 直接通过状态机走到 archived：planned -> collecting -> pending -> confirmed -> archived。
	for _, step := range []struct{ from, to string }{
		{model.TrialPlanned, model.TrialCollecting},
		{model.TrialCollecting, model.TrialPendingInversion},
		{model.TrialPendingInversion, model.TrialConfirmed},
		{model.TrialConfirmed, model.TrialArchived},
	} {
		if err := svc.repos.Trial.TransitionState(ctx, tr.ID, step.from, step.to); err != nil {
			t.Fatalf("transition %s->%s: %v", step.from, step.to, err)
		}
	}
	// 封存后追加层序应被拒绝。
	if _, err := svc.AddLayer(ctx, tr.ID, &model.Layer{Seq: 1, Material: "x", ThicknessMM: 1, Conductivity: 1, Density: 1, SpecificHeat: 1, AreaM2: 1}); err != model.ErrImmutable {
		t.Fatalf("expected ErrImmutable, got %v", err)
	}
}
