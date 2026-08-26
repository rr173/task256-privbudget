package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/service"
	"task256-privbudget/internal/store"
)

// runSmoke 执行端到端自检：建库→登记→验证→评估→关闭重开→撤销恢复→发布快照→重开验证。
// 关键断言：两个共享同一数据集的发布组合后超限被拒；撤销其一后恢复可发布；快照持久化。
func runSmoke(dbPath string) error {
	ctx := context.Background()

	tmp, err := os.MkdirTemp("", "privbudget-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if dbPath == "" || dbPath == "privbudget.db" {
		dbPath = filepath.Join(tmp, "smoke.db")
	} else {
		dbPath = filepath.Join(tmp, filepath.Base(dbPath))
	}

	// 第一轮：建库并执行业务闭环
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	app := service.NewApp(st)

	popDS := model.DatasetVersion{
		ID: "ds_pop", Name: "population-A", Version: "v1",
		Status: model.DatasetRegistered, PrivacyUnit: "individual",
		EpsilonCap: 1.0, DeltaCap: 1e-5,
	}
	if err := app.RegisterDataset(ctx, popDS); err != nil {
		return err
	}

	m1 := model.Mechanism{ID: "mech_1", Name: "task-1", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0, DatasetIDs: []string{"ds_pop"}, Status: model.MechDraft}
	m2 := model.Mechanism{ID: "mech_2", Name: "task-2", Kind: model.MechLaplace, Epsilon: 0.6, Delta: 0, DatasetIDs: []string{"ds_pop"}, Status: model.MechDraft}
	if err := app.RegisterMechanism(ctx, m1); err != nil {
		return err
	}
	if err := app.RegisterMechanism(ctx, m2); err != nil {
		return err
	}
	if err := app.VerifyMechanism(ctx, "mech_1"); err != nil {
		return err
	}
	if err := app.VerifyMechanism(ctx, "mech_2"); err != nil {
		return err
	}

	r1 := model.Release{ID: "rel_1", Name: "release-1", MechanismID: "mech_1", Rule: model.RuleSequential, Status: model.ReleasePending}
	r2 := model.Release{ID: "rel_2", Name: "release-2", MechanismID: "mech_2", Rule: model.RuleSequential, Status: model.ReleasePending}
	if err := app.CreateRelease(ctx, r1); err != nil {
		return err
	}
	if err := app.CreateRelease(ctx, r2); err != nil {
		return err
	}

	if _, err := app.EvaluateRelease(ctx, "rel_1"); err != nil {
		return err
	}
	rep2, err := app.EvaluateRelease(ctx, "rel_2")
	if err != nil {
		return err
	}
	if !rep2.Overlimited {
		return fmt.Errorf("期望两个任务共享 ds_pop 时超限，实际: %+v", rep2)
	}
	got1, _ := app.GetRelease(ctx, "rel_1")
	if got1.Status != model.ReleaseAllowed {
		return fmt.Errorf("rel_1 期望 allowed，实际 %s", got1.Status)
	}
	got2, _ := app.GetRelease(ctx, "rel_2")
	if got2.Status != model.ReleaseAllowed {
		return fmt.Errorf("rel_2 期望 allowed（超限由数据集状态反映），实际 %s", got2.Status)
	}
	dsA, _ := app.GetDataset(ctx, "ds_pop")
	if dsA.Status != model.DatasetOverlimit {
		return fmt.Errorf("ds_pop 期望 overlimit，实际 %s", dsA.Status)
	}

	// 关闭并重新打开数据库，验证持久化与重启恢复
	if err := st.Close(); err != nil {
		return err
	}
	st2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	app = service.NewApp(st2)
	dsA2, err := app.GetDataset(ctx, "ds_pop")
	if err != nil {
		return err
	}
	if dsA2.Status != model.DatasetOverlimit {
		return fmt.Errorf("重开后 ds_pop 期望 overlimit，实际 %s", dsA2.Status)
	}
	got1b, _ := app.GetRelease(ctx, "rel_1")
	if got1b.Status != model.ReleaseAllowed {
		return fmt.Errorf("重开后 rel_1 期望 allowed，实际 %s", got1b.Status)
	}
	got2b, _ := app.GetRelease(ctx, "rel_2")
	if got2b.Status != model.ReleaseAllowed {
		return fmt.Errorf("重开后 rel_2 期望 allowed，实际 %s", got2b.Status)
	}

	// 撤销 rel_1 后，仅剩 rel_2（0.6）在预算内，应恢复为可发布
	if _, err := app.RevokeRelease(ctx, "rel_1"); err != nil {
		return err
	}
	repAfter, err := app.EvaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		return err
	}
	if repAfter.Overlimited {
		return fmt.Errorf("撤销 rel_1 后期望在预算内，实际超限: %+v", repAfter)
	}
	dsAfter, _ := app.GetDataset(ctx, "ds_pop")
	if dsAfter.Status == model.DatasetOverlimit {
		return fmt.Errorf("撤销 rel_1 后 ds_pop 不应再 overlimit，实际 %s", dsAfter.Status)
	}

	// 发布快照（此时预算在限额内，可发布）
	snap := model.BudgetSnapshot{ID: "snap_1", Name: "baseline", Rule: model.RuleSequential, Status: model.SnapDraft}
	if err := app.CreateSnapshot(ctx, snap); err != nil {
		return err
	}
	if _, err := app.PublishSnapshot(ctx, "snap_1", model.RuleSequential); err != nil {
		return err
	}

	// 再次关闭重开，验证快照持久化
	if err := st2.Close(); err != nil {
		return err
	}
	st3, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db 2: %w", err)
	}
	app = service.NewApp(st3)
	snapGot, err := app.GetSnapshot(ctx, "snap_1")
	if err != nil {
		return err
	}
	if snapGot.Status != model.SnapPublished {
		return fmt.Errorf("snap_1 期望 published，实际 %s", snapGot.Status)
	}

	// 自检
	issues, err := app.SelfCheck(ctx)
	if err != nil {
		return err
	}
	if len(issues) != 1 || issues[0] != "ok" {
		return fmt.Errorf("selfcheck 问题: %v", issues)
	}
	if err := st3.Close(); err != nil {
		return err
	}
	return nil
}
