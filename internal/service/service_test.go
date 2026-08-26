package service

import (
	"context"
	"errors"
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

// TestEvaluateReleaseSealedDatasetBoundary 验证封存发布边界：
// 数据集封存后，即便引用它的机制仍有效，对其产生的新发布评估必须失败，
// 发布保持待评估状态，且已有预算与历史发布不被改变。
func TestEvaluateReleaseSealedDatasetBoundary(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(st)

	pop := model.DatasetVersion{ID: "ds_seal", Name: "pop", Version: "v1", Status: model.DatasetRegistered, EpsilonCap: 1.0, DeltaCap: 1e-5}
	if err := app.RegisterDataset(ctx, pop); err != nil {
		t.Fatal(err)
	}
	m := model.Mechanism{ID: "m_seal", Kind: model.MechLaplace, Epsilon: 0.3, Delta: 0, DatasetIDs: []string{"ds_seal"}, Status: model.MechDraft}
	if err := app.RegisterMechanism(ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyMechanism(ctx, "m_seal"); err != nil {
		t.Fatal(err)
	}

	// 历史发布 r_prev：封存前评估通过，作为后续断言“历史发布不被改变”的基线。
	rPrev := model.Release{ID: "r_prev", MechanismID: "m_seal", Rule: model.RuleSequential, Status: model.ReleasePending}
	if err := app.CreateRelease(ctx, rPrev); err != nil {
		t.Fatal(err)
	}
	if _, err := app.EvaluateRelease(ctx, "r_prev"); err != nil {
		t.Fatal(err)
	}
	rPrevGot, _ := app.GetRelease(ctx, "r_prev")
	if rPrevGot.Status != model.ReleaseAllowed {
		t.Fatalf("r_prev expected allowed, got %s", rPrevGot.Status)
	}

	// 封存数据集
	if err := app.SealDataset(ctx, "ds_seal"); err != nil {
		t.Fatal(err)
	}

	// 对同一有效机制产生的新发布：评估必须失败
	rAfter := model.Release{ID: "r_after", MechanismID: "m_seal", Rule: model.RuleSequential, Status: model.ReleasePending}
	if err := app.CreateRelease(ctx, rAfter); err != nil {
		t.Fatal(err)
	}
	if _, err := app.EvaluateRelease(ctx, "r_after"); !errors.Is(err, model.ErrDatasetSealed) {
		t.Fatalf("expected ErrDatasetSealed after sealing dataset, got %v", err)
	}

	// 发布仍保持待评估状态（未设置评估时间、未推进为 allowed/rejected）
	rAfterGot, _ := app.GetRelease(ctx, "r_after")
	if rAfterGot.Status != model.ReleasePending {
		t.Fatalf("r_after expected pending after failed evaluation, got %s", rAfterGot.Status)
	}
	if rAfterGot.EvaluatedAt != nil {
		t.Fatalf("r_after expected no evaluated_at, got %v", rAfterGot.EvaluatedAt)
	}

	// 历史发布 r_prev 不被改变（封存不应回溯影响已发布的发布）
	rPrevGot2, _ := app.GetRelease(ctx, "r_prev")
	if rPrevGot2.Status != model.ReleaseAllowed {
		t.Fatalf("r_prev expected to remain allowed, got %s", rPrevGot2.Status)
	}

	// 已有预算不受影响：r_prev 的消耗仍计入，但不超限（0.3 ≤ 1.0）
	rep, err := app.EvaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Overlimited {
		t.Fatalf("budget should remain within limits, got overlimit: %+v", rep.Entries)
	}
}
