// Package mechanism 统计机制模块：维护差分隐私机制的参数登记、验证与撤销。
package mechanism

import (
	"context"
	"fmt"
	"time"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/store"
)

// Service 统计机制相关业务逻辑。
type Service struct {
	store *store.Store
}

// New 构造机制服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// Register 登记一个统计机制（参数非法直接报错，不落库）。
func (s *Service) Register(ctx context.Context, m model.Mechanism) error {
	if m.ID == "" {
		return fmt.Errorf("mechanism id required")
	}
	if err := validateParams(m); err != nil {
		return err
	}
	if m.Status == "" {
		m.Status = model.MechDraft
	}
	// 规范化数据集声明：去重，避免重复声明在预算计费时被放大。
	m.DatasetIDs = model.DedupDatasetIDs(m.DatasetIDs)
	return s.store.MechanismCreate(ctx, m)
}

// Get 按 ID 取机制。
func (s *Service) Get(ctx context.Context, id string) (model.Mechanism, error) {
	return s.store.MechanismGet(ctx, id)
}

// List 列出全部机制。
func (s *Service) List(ctx context.Context) ([]model.Mechanism, error) {
	return s.store.MechanismList(ctx)
}

// Verify 校验机制参数并确认其引用的数据集均存在，通过后置为已验证。
func (s *Service) Verify(ctx context.Context, id string) error {
	m, err := s.store.MechanismGet(ctx, id)
	if err != nil {
		return err
	}
	if err := validateParams(m); err != nil {
		return err
	}
	all, err := s.store.DatasetList(ctx)
	if err != nil {
		return err
	}
	byID := make(map[string]bool, len(all))
	for _, d := range all {
		byID[d.ID] = true
	}
	for _, did := range m.DatasetIDs {
		if !byID[did] {
			return model.ErrDatasetMissing
		}
	}
	now := time.Now().UTC()
	m.Status = model.MechVerified
	m.ValidatedAt = &now
	return s.store.MechanismUpdate(ctx, m)
}

// Revoke 撤销机制（使其不再作为有效预算来源）。
func (s *Service) Revoke(ctx context.Context, id string) error {
	m, err := s.store.MechanismGet(ctx, id)
	if err != nil {
		return err
	}
	m.Status = model.MechRevoked
	return s.store.MechanismUpdate(ctx, m)
}
