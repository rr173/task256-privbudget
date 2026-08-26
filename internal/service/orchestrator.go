// Package service 编排层：组合数据集、机制、发布、快照四个业务模块，
// 维护全局写锁以保证预算扣减的串行一致性，并在每次变更后刷新数据集的派生状态、
// 提供预算评估与自检能力。
package service

import (
	"context"
	"fmt"
	"sync"

	"task256-privbudget/internal/compose"
	"task256-privbudget/internal/dataset"
	"task256-privbudget/internal/mechanism"
	"task256-privbudget/internal/model"
	"task256-privbudget/internal/release"
	"task256-privbudget/internal/snapshot"
	"task256-privbudget/internal/store"
)

// RuleVersion 组合规则版本号，随快照冻结。
const RuleVersion = "dp-compose-2026.1"

// App 顶层应用服务。
type App struct {
	store    *store.Store
	datasets *dataset.Service
	mechs    *mechanism.Service
	releases *release.Service
	snaps    *snapshot.Service
	mu       sync.Mutex
}

// NewApp 构造应用服务。
func NewApp(st *store.Store) *App {
	return &App{
		store:    st,
		datasets: dataset.New(st),
		mechs:    mechanism.New(st),
		releases: release.New(st),
		snaps:    snapshot.New(st),
	}
}

// Store 暴露底层存储（自检/测试用）。
func (a *App) Store() *store.Store { return a.store }

// world 构建当前世界快照（调用方须已持有锁）。
func (a *App) world(ctx context.Context) (compose.World, error) {
	ds, err := a.store.DatasetList(ctx)
	if err != nil {
		return compose.World{}, err
	}
	ms, err := a.store.MechanismList(ctx)
	if err != nil {
		return compose.World{}, err
	}
	rs, err := a.store.ReleaseList(ctx)
	if err != nil {
		return compose.World{}, err
	}
	return compose.World{Datasets: ds, Mechanisms: ms, Releases: rs}, nil
}

// evaluateBudget 计算预算结论（调用方须已持有锁）。
func (a *App) evaluateBudget(ctx context.Context, rule model.CompositionRule) (*compose.Report, error) {
	w, err := a.world(ctx)
	if err != nil {
		return nil, err
	}
	return compose.Evaluate(w, rule)
}

// ---- 数据集 ----

// RegisterDataset 登记数据集版本。
func (a *App) RegisterDataset(ctx context.Context, d model.DatasetVersion) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.datasets.Register(ctx, d)
}

// GetDataset 取数据集版本。
func (a *App) GetDataset(ctx context.Context, id string) (model.DatasetVersion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.datasets.Get(ctx, id)
}

// ListDatasets 列出数据集版本。
func (a *App) ListDatasets(ctx context.Context) ([]model.DatasetVersion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.datasets.List(ctx)
}

// UpdateDataset 更新数据集版本。
func (a *App) UpdateDataset(ctx context.Context, d model.DatasetVersion) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.datasets.Update(ctx, d)
}

// SealDataset 封存数据集版本。
func (a *App) SealDataset(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.datasets.Seal(ctx, id)
}

// ---- 机制 ----

// RegisterMechanism 登记机制。
func (a *App) RegisterMechanism(ctx context.Context, m model.Mechanism) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mechs.Register(ctx, m)
}

// VerifyMechanism 验证机制。
func (a *App) VerifyMechanism(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mechs.Verify(ctx, id)
}

// RevokeMechanism 撤销机制。
func (a *App) RevokeMechanism(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mechs.Revoke(ctx, id)
}

// GetMechanism 取机制。
func (a *App) GetMechanism(ctx context.Context, id string) (model.Mechanism, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mechs.Get(ctx, id)
}

// ListMechanisms 列出机制。
func (a *App) ListMechanisms(ctx context.Context) ([]model.Mechanism, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mechs.List(ctx)
}

// ---- 发布 ----

// CreateRelease 创建发布批次。
func (a *App) CreateRelease(ctx context.Context, r model.Release) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.releases.Create(ctx, r)
}

// EvaluateRelease 将待评估发布推进到允许/拒绝，并刷新预算状态。返回的 report
// 为“把该发布纳入后进的预算结论”。
func (a *App) EvaluateRelease(ctx context.Context, releaseID string) (*compose.Report, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	r, err := a.store.ReleaseGet(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	if r.Status != model.ReleasePending {
		return nil, model.ErrReleaseConflict
	}
	w, err := a.world(ctx)
	if err != nil {
		return nil, err
	}
	// 将待评估发布临时置为 allowed 以测试其影响
	for i := range w.Releases {
		if w.Releases[i].ID == releaseID {
			w.Releases[i].Status = model.ReleaseAllowed
		}
	}
	rep, err := compose.Evaluate(w, r.Rule)
	if err != nil {
		return nil, err
	}
	m, err := a.store.MechanismGet(ctx, r.MechanismID)
	if err != nil {
		return nil, err
	}
	decided := release.Decide(r, m, rep)
	if err := a.releases.ApplyDecision(ctx, decided); err != nil {
		return nil, err
	}
	if err := a.refreshStatuses(ctx); err != nil {
		return nil, err
	}
	return rep, nil
}

// RevokeRelease 撤销已允许的发布并刷新预算状态。
func (a *App) RevokeRelease(ctx context.Context, releaseID string) (*compose.Report, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.releases.Revoke(ctx, releaseID); err != nil {
		return nil, err
	}
	if err := a.refreshStatuses(ctx); err != nil {
		return nil, err
	}
	return a.evaluateBudget(ctx, model.RuleSequential)
}

// GetRelease 取发布。
func (a *App) GetRelease(ctx context.Context, id string) (model.Release, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.releases.Get(ctx, id)
}

// ListReleases 列出发布。
func (a *App) ListReleases(ctx context.Context) ([]model.Release, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.releases.List(ctx)
}

// ---- 预算评估 ----

// EvaluateBudget 评估当前世界的预算结论（默认规则 sequential）。
func (a *App) EvaluateBudget(ctx context.Context, rule model.CompositionRule) (*compose.Report, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.evaluateBudget(ctx, rule)
}

// refreshStatuses 依据顺序组合结论刷新各数据集的派生状态（封存者除外）。
func (a *App) refreshStatuses(ctx context.Context) error {
	w, err := a.world(ctx)
	if err != nil {
		return err
	}
	rep, err := compose.Evaluate(w, model.RuleSequential)
	if err != nil {
		return err
	}
	rootEntry := make(map[string]model.BudgetEntry, len(rep.Entries))
	for _, e := range rep.Entries {
		rootEntry[e.Key] = e
	}
	g, err := compose.BuildGraph(w.Datasets)
	if err != nil {
		return err
	}
	for _, d := range w.Datasets {
		if d.IsSealed() {
			continue
		}
		root := g.RootOf(d.ID)
		entry, ok := rootEntry[root]
		over := ok && (entry.EpsilonUsed > entry.EpsilonCap+1e-9 || entry.DeltaUsed > entry.DeltaCap+1e-9)
		near := false
		if ok {
			if entry.EpsilonCap > 0 && entry.EpsilonUsed >= 0.8*entry.EpsilonCap {
				near = true
			}
			if entry.DeltaCap > 0 && entry.DeltaUsed >= 0.8*entry.DeltaCap {
				near = true
			}
		}
		var st model.DatasetStatus
		switch {
		case over:
			st = model.DatasetOverlimit
		case near:
			st = model.DatasetTense
		default:
			st = model.DatasetReleasable
		}
		if d.Status != st {
			d.Status = st
			if err := a.store.DatasetUpdate(ctx, d); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---- 快照 ----

// CreateSnapshot 创建草稿快照。
func (a *App) CreateSnapshot(ctx context.Context, snap model.BudgetSnapshot) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snaps.CreateDraft(ctx, snap)
}

// PublishSnapshot 将草稿快照冻结为已发布，返回冻结所用的预算结论。
func (a *App) PublishSnapshot(ctx context.Context, id string, rule model.CompositionRule) (*compose.Report, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	rep, err := a.evaluateBudget(ctx, rule)
	if err != nil {
		return nil, err
	}
	entries := append([]model.BudgetEntry(nil), rep.Entries...)
	summary := snapshot.BuildSummary(rep)
	if err := a.snaps.Publish(ctx, id, entries, summary, RuleVersion); err != nil {
		return nil, err
	}
	return rep, nil
}

// GetSnapshot 取快照。
func (a *App) GetSnapshot(ctx context.Context, id string) (model.BudgetSnapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snaps.Get(ctx, id)
}

// ListSnapshots 列出快照。
func (a *App) ListSnapshots(ctx context.Context) ([]model.BudgetSnapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snaps.List(ctx)
}

// ---- 自检 ----

// SelfCheck 运行内部不变量检查，返回问题列表（无问题则含 "ok"）。
func (a *App) SelfCheck(ctx context.Context) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var issues []string
	w, err := a.world(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := compose.BuildGraph(w.Datasets); err != nil {
		issues = append(issues, "dataset derivation cycle: "+err.Error())
	}
	mechByID := make(map[string]model.Mechanism, len(w.Mechanisms))
	for _, m := range w.Mechanisms {
		mechByID[m.ID] = m
	}
	for _, r := range w.Releases {
		if r.Status != model.ReleaseAllowed {
			continue
		}
		m, ok := mechByID[r.MechanismID]
		if !ok {
			issues = append(issues, fmt.Sprintf("release %s references missing mechanism", r.ID))
			continue
		}
		if !m.IsLive() {
			issues = append(issues, fmt.Sprintf("release %s references non-live mechanism (%s)", r.ID, m.Status))
		}
	}
	for _, m := range w.Mechanisms {
		if m.Epsilon <= 0 || m.Delta < 0 || m.Delta >= 1 {
			issues = append(issues, fmt.Sprintf("mechanism %s has illegal epsilon/delta", m.ID))
		}
	}
	rep1, err := a.evaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		return nil, err
	}
	rep2, err := a.evaluateBudget(ctx, model.RuleSequential)
	if err != nil {
		return nil, err
	}
	if !sameReport(rep1, rep2) {
		issues = append(issues, "budget evaluation is not idempotent")
	}
	if len(issues) == 0 {
		issues = append(issues, "ok")
	}
	return issues, nil
}

// sameReport 比较两个评估结论是否一致（用于幂等自检）。
func sameReport(a, b *compose.Report) bool {
	if a.Overlimited != b.Overlimited || len(a.Entries) != len(b.Entries) {
		return false
	}
	key := func(e model.BudgetEntry) string {
		return fmt.Sprintf("%s|%s|%.6f|%.6f|%v", e.Key, e.Scope, e.EpsilonUsed, e.DeltaUsed, e.Overlimit)
	}
	seen := map[string]int{}
	for _, e := range a.Entries {
		seen[key(e)]++
	}
	for _, e := range b.Entries {
		seen[key(e)]--
		if seen[key(e)] < 0 {
			return false
		}
	}
	for _, v := range seen {
		if v != 0 {
			return false
		}
	}
	return true
}
