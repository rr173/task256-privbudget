package service

import (
	"context"
	"path/filepath"
	"testing"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/store"
)

// TestCreateReleaseIgnoresAllowedStatus 回归：客户端在创建请求里带 "allowed"
// 状态也不能让新发布直接计入预算消耗。新发布必须先 pending，评估通过后才
// 进入 allowed。否则客户端可凭请求体绕过评估直接花预算。
func TestCreateReleaseIgnoresAllowedStatus(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(st)

	pop := model.DatasetVersion{
		ID: "ds", Name: "pop", Version: "v1",
		Status: model.DatasetRegistered, EpsilonCap: 1.0, DeltaCap: 1e-5,
	}
	if err := app.RegisterDataset(ctx, pop); err != nil {
		t.Fatal(err)
	}
	m := model.Mechanism{
		ID: "m1", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0,
		DatasetIDs: []string{"ds"}, Status: model.MechDraft,
	}
	if err := app.RegisterMechanism(ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := app.VerifyMechanism(ctx, "m1"); err != nil {
		t.Fatal(err)
	}

	// 客户端恶意/错误地带上 allowed 状态与已评估字段，期望服务端一律忽略。
	malicious := model.Release{
		ID: "r1", MechanismID: "m1", Rule: model.RuleSequential,
		Status: model.ReleaseAllowed, // 试图直接 allowed
	}
	if err := app.CreateRelease(ctx, malicious); err != nil {
		t.Fatal(err)
	}
	got, err := app.GetRelease(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.ReleasePending {
		t.Fatalf("new release must be pending, got %s", got.Status)
	}
	if got.EvaluatedAt != nil {
		t.Fatalf("new release must not be evaluated, got evaluated_at=%v", got.EvaluatedAt)
	}
	if !got.IsLive() {
		// IsLive()==false 正是期望：未评估发布不计入实时预算。
	} else {
		t.Fatalf("unevaluated release must not be live")
	}

	// 未经评估的发布不应消耗任何预算：消耗的 ε 应为 0。
	rep, err := app.EvaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		t.Fatal(err)
	}
	var epsUsed float64
	for _, e := range rep.Entries {
		if e.Key == "ds" {
			epsUsed = e.EpsilonUsed
		}
	}
	if epsUsed != 0 {
		t.Fatalf("unevaluated release must not consume budget, got epsilon_used=%.4f", epsUsed)
	}

	// 评估通过后才计入预算。
	if _, err := app.EvaluateRelease(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
	evaluated, err := app.GetRelease(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if evaluated.Status != model.ReleaseAllowed || !evaluated.IsLive() {
		t.Fatalf("after evaluation expected allowed/live, got %s", evaluated.Status)
	}
	if evaluated.EvaluatedAt == nil {
		t.Fatalf("evaluated release must carry evaluated_at")
	}
	repAfter, err := app.EvaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		t.Fatal(err)
	}
	var epsAfter float64
	for _, e := range repAfter.Entries {
		if e.Key == "ds" {
			epsAfter = e.EpsilonUsed
		}
	}
	if epsAfter == 0 {
		t.Fatalf("allowed release must consume budget, got epsilon_used=0")
	}
}
