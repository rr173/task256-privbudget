// Package snapshot 预算快照模块：冻结某一时刻的预算组合结论。
package snapshot

import (
	"context"
	"fmt"

	"task256-privbudget/internal/compose"
	"task256-privbudget/internal/model"
	"task256-privbudget/internal/store"
)

// Service 预算快照相关业务逻辑。
type Service struct {
	store *store.Store
}

// New 构造快照服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// CreateDraft 创建草稿快照（幂等）。
func (s *Service) CreateDraft(ctx context.Context, snap model.BudgetSnapshot) error {
	if snap.ID == "" {
		return fmt.Errorf("snapshot id required")
	}
	if snap.Status == "" {
		snap.Status = model.SnapDraft
	}
	return s.store.SnapshotCreate(ctx, snap)
}

// Publish 发布草稿快照：冻结预算单元、摘要与规则版本，并将此前已发布者置为替代。
func (s *Service) Publish(ctx context.Context, id string, entries []model.BudgetEntry, summary, ruleVersion string) error {
	snap, err := s.store.SnapshotGet(ctx, id)
	if err != nil {
		return err
	}
	if snap.Status != model.SnapDraft {
		return model.ErrSnapshotSealed
	}
	if err := s.supersedePublished(ctx); err != nil {
		return err
	}
	now := snap.FrozenAt
	if now.IsZero() {
		now = snap.CreatedAt
	}
	snap.Status = model.SnapPublished
	snap.Entries = entries
	snap.Summary = summary
	snap.RuleVersion = ruleVersion
	snap.FrozenAt = now
	return s.store.SnapshotUpdate(ctx, snap)
}

// supersedePublished 将此前已发布的快照降级为替代。
func (s *Service) supersedePublished(ctx context.Context) error {
	all, err := s.store.SnapshotList(ctx)
	if err != nil {
		return err
	}
	for _, sn := range all {
		if sn.Status == model.SnapPublished {
			sn.Status = model.SnapSuperseded
			if err := s.store.SnapshotUpdate(ctx, sn); err != nil {
				return err
			}
		}
	}
	return nil
}

// Get 按 ID 取快照。
func (s *Service) Get(ctx context.Context, id string) (model.BudgetSnapshot, error) {
	return s.store.SnapshotGet(ctx, id)
}

// List 列出全部快照。
func (s *Service) List(ctx context.Context) ([]model.BudgetSnapshot, error) {
	return s.store.SnapshotList(ctx)
}

// BuildSummary 生成人类可读的结论摘要。
func BuildSummary(rep *compose.Report) string {
	if !rep.Overlimited {
		return "budget within limits: all population units satisfied"
	}
	n := 0
	for _, e := range rep.Entries {
		if e.Overlimit {
			n++
		}
	}
	return fmt.Sprintf("budget OVERLIMIT: %d unit(s) exceed cap under rule %s", n, rep.Rule)
}
