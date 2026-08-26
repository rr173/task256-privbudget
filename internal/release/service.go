// Package release 发布批次模块：维护统计发布的创建、评估决定与撤销回退。
package release

import (
	"context"
	"fmt"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/store"
)

// Service 发布批次相关业务逻辑。
type Service struct {
	store *store.Store
}

// New 构造发布服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// knownRule 判断组合规则是否受支持。
func knownRule(rule model.CompositionRule) bool {
	switch rule {
	case model.RuleSequential, model.RuleAdvanced, model.RuleParallel, model.RuleRDP:
		return true
	}
	return false
}

// Create 创建发布批次（默认 pending），校验引用的机制存在且规则合法。
func (s *Service) Create(ctx context.Context, r model.Release) error {
	if r.ID == "" {
		return fmt.Errorf("release id required")
	}
	if !knownRule(r.Rule) {
		return model.ErrUnknownRule
	}
	if _, err := s.store.MechanismGet(ctx, r.MechanismID); err != nil {
		return model.ErrMechanismMissing
	}
	if r.Status == "" {
		r.Status = model.ReleasePending
	}
	return s.store.ReleaseCreate(ctx, r)
}

// Get 按 ID 取发布。
func (s *Service) Get(ctx context.Context, id string) (model.Release, error) {
	return s.store.ReleaseGet(ctx, id)
}

// List 列出全部发布。
func (s *Service) List(ctx context.Context) ([]model.Release, error) {
	return s.store.ReleaseList(ctx)
}

// ApplyDecision 将评估决定（状态、超限路径、评估时间）写回存储。
func (s *Service) ApplyDecision(ctx context.Context, r model.Release) error {
	return s.store.ReleaseUpdate(ctx, r)
}

// Revoke 撤销一个已允许的发布，回退其预算消耗。
func (s *Service) Revoke(ctx context.Context, id string) error {
	r, err := s.store.ReleaseGet(ctx, id)
	if err != nil {
		return err
	}
	if r.Status != model.ReleaseAllowed {
		return model.ErrRevokeMissing
	}
	r.Status = model.ReleaseRevoked
	return s.store.ReleaseUpdate(ctx, r)
}
