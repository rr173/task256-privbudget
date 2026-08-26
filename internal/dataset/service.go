// Package dataset 数据集版本模块：维护版本化数据集及其派生 DAG 的登记与冻结。
package dataset

import (
	"context"
	"fmt"
	"time"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/store"
)

// Service 数据集版本相关业务逻辑。
type Service struct {
	store *store.Store
}

// New 构造数据集服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// Register 登记一个数据集版本（幂等：重复 ID 报 ErrAlreadyExists）。
func (s *Service) Register(ctx context.Context, d model.DatasetVersion) error {
	if d.ID == "" {
		return fmt.Errorf("dataset id required")
	}
	if d.EpsilonCap < 0 || d.DeltaCap < 0 {
		return model.ErrIllegalEpsilonDelta
	}
	if d.Status == "" {
		d.Status = model.DatasetRegistered
	}
	if err := s.checkParents(ctx, d); err != nil {
		return err
	}
	return s.store.DatasetCreate(ctx, d)
}

// Get 按 ID 取数据集版本。
func (s *Service) Get(ctx context.Context, id string) (model.DatasetVersion, error) {
	return s.store.DatasetGet(ctx, id)
}

// List 列出全部数据集版本。
func (s *Service) List(ctx context.Context) ([]model.DatasetVersion, error) {
	return s.store.DatasetList(ctx)
}

// Update 更新数据集版本（已封存者禁止修改）。
func (s *Service) Update(ctx context.Context, d model.DatasetVersion) error {
	cur, err := s.store.DatasetGet(ctx, d.ID)
	if err != nil {
		return err
	}
	if cur.IsSealed() {
		return model.ErrDatasetSealed
	}
	if d.EpsilonCap < 0 || d.DeltaCap < 0 {
		return model.ErrIllegalEpsilonDelta
	}
	if err := s.checkParents(ctx, d); err != nil {
		return err
	}
	return s.store.DatasetUpdate(ctx, d)
}

// Seal 封存数据集版本（冻结，禁止后续修改），幂等。
func (s *Service) Seal(ctx context.Context, id string) error {
	cur, err := s.store.DatasetGet(ctx, id)
	if err != nil {
		return err
	}
	if cur.IsSealed() {
		return nil
	}
	now := time.Now().UTC()
	cur.Status = model.DatasetSealed
	cur.SealedAt = &now
	return s.store.DatasetUpdate(ctx, cur)
}
