package service

import (
	"context"
	"path/filepath"
	"testing"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/store"
)

// TestServiceIntegration 走通“登记→验证→评估→撤销→快照→自检”闭环并验证持久化。
func TestServiceIntegration(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(st)

	pop := model.DatasetVersion{ID: "ds", Name: "pop", Version: "v1", Status: model.DatasetRegistered, EpsilonCap: 1.0, DeltaCap: 1e-5}
	if err := app.RegisterDataset(ctx, pop); err != nil {
		t.Fatal(err)
	}
	m := model.Mechanism{ID: "m1", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0, DatasetIDs: []string{"ds"}, Status: model.MechDraft}
	if err := app.RegisterMechanism(ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyMechanism(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	rel := model.Release{ID: "r1", MechanismID: "m1", Rule: model.RuleSequential, Status: model.ReleasePending}
	if err := app.CreateRelease(ctx, rel); err != nil {
		t.Fatal(err)
	}
	if _, err := app.EvaluateRelease(ctx, "r1"); err != nil {
		t.Fatal(err)
	}

	// 第二个共享同一数据集的发布 → 组合超限
	m2 := model.Mechanism{ID: "m2", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0, DatasetIDs: []string{"ds"}, Status: model.MechDraft}
	if err := app.RegisterMechanism(ctx, m2); err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyMechanism(ctx, "m2"); err != nil {
		t.Fatal(err)
	}
	rel2 := model.Release{ID: "r2", MechanismID: "m2", Rule: model.RuleSequential, Status: model.ReleasePending}
	if err := app.CreateRelease(ctx, rel2); err != nil {
		t.Fatal(err)
	}
	rep, err := app.EvaluateRelease(ctx, "r2")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Overlimited {
		t.Fatalf("expected overlimit, got %+v", rep.Entries)
	}
	dsGot, _ := app.GetDataset(ctx, "ds")
	if dsGot.Status != model.DatasetOverlimit {
		t.Fatalf("ds expected overlimit, got %s", dsGot.Status)
	}

	// 撤销 r1 → 恢复
	if _, err := app.RevokeRelease(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
	repAfter, err := app.EvaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		t.Fatal(err)
	}
	if repAfter.Overlimited {
		t.Fatalf("after revoke expected within limits, got %+v", repAfter.Entries)
	}

	// 发布快照
	snap := model.BudgetSnapshot{ID: "s1", Name: "snap", Rule: model.RuleSequential, Status: model.SnapDraft}
	if err := app.CreateSnapshot(ctx, snap); err != nil {
		t.Fatal(err)
	}
	if _, err := app.PublishSnapshot(ctx, "s1", model.RuleSequential); err != nil {
		t.Fatal(err)
	}
	snapGot, err := app.GetSnapshot(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if snapGot.Status != model.SnapPublished {
		t.Fatalf("snap expected published, got %s", snapGot.Status)
	}

	// 自检
	issues, err := app.SelfCheck(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0] != "ok" {
		t.Fatalf("selfcheck issues: %v", issues)
	}
}
