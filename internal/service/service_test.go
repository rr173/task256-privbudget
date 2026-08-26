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

// TestRevokeMechanismRestoresBudget 验证：撤销一个被已允许发布所使用的统计机制后，
// 该机制不再计入实时预算，且此前因该机制贡献而被判为超限的未封存数据集应立即
// 恢复到正确的预算状态（而非停留在陈旧的 overlimit）。
func TestRevokeMechanismRestoresBudget(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(st)

	if err := app.RegisterDataset(ctx, model.DatasetVersion{
		ID: "ds", Name: "pop", Version: "v1", Status: model.DatasetRegistered,
		EpsilonCap: 1.0, DeltaCap: 1e-5,
	}); err != nil {
		t.Fatal(err)
	}
	// 两个机制各 0.6ε，作用于同一数据集——组合后 1.2ε > 1.0 上限即超限。
	m1 := model.Mechanism{ID: "m1", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0, DatasetIDs: []string{"ds"}, Status: model.MechDraft}
	m2 := model.Mechanism{ID: "m2", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0, DatasetIDs: []string{"ds"}, Status: model.MechDraft}
	if err := app.RegisterMechanism(ctx, m1); err != nil {
		t.Fatal(err)
	}
	if err := app.RegisterMechanism(ctx, m2); err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyMechanism(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyMechanism(ctx, "m2"); err != nil {
		t.Fatal(err)
	}
	r1 := model.Release{ID: "r1", MechanismID: "m1", Rule: model.RuleSequential, Status: model.ReleasePending}
	r2 := model.Release{ID: "r2", MechanismID: "m2", Rule: model.RuleSequential, Status: model.ReleasePending}
	if err := app.CreateRelease(ctx, r1); err != nil {
		t.Fatal(err)
	}
	if err := app.CreateRelease(ctx, r2); err != nil {
		t.Fatal(err)
	}
	if _, err := app.EvaluateRelease(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.EvaluateRelease(ctx, "r2"); err != nil {
		t.Fatal(err)
	}
	// 组合后超限：数据集应处于 overlimit。
	dsBefore, _ := app.GetDataset(ctx, "ds")
	if dsBefore.Status != model.DatasetOverlimit {
		t.Fatalf("ds expected overlimit before revoke, got %s", dsBefore.Status)
	}

	// 撤销 m2：其实时预算贡献被移除，仅剩 r1（0.6ε），应在预算内。
	if err := app.RevokeMechanism(ctx, "m2"); err != nil {
		t.Fatal(err)
	}

	// 实时预算评估不应再超限。
	rep, err := app.EvaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Overlimited {
		t.Fatalf("after revoking m2 expected within limits, got %+v", rep.Entries)
	}
	// 未封存数据集应已随之刷新为可发布（而非停留在陈旧的 overlimit）。
	dsAfter, _ := app.GetDataset(ctx, "ds")
	if dsAfter.Status == model.DatasetOverlimit {
		t.Fatalf("after revoking m2 ds should not remain overlimit, got %s", dsAfter.Status)
	}
}
