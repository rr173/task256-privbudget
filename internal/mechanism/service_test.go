package mechanism

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"task256-privbudget/internal/model"
	"task256-privbudget/internal/store"
)

// newService 构造基于临时 SQLite 库的机制服务与数据集服务。
func newService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "mech.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st), st
}

func newLiveMech(id string, dsIDs ...string) model.Mechanism {
	return model.Mechanism{
		ID: id, Name: id, Kind: model.MechLaplace,
		Epsilon: 0.5, Delta: 0, DatasetIDs: dsIDs, Status: model.MechDraft,
	}
}

// TestRegisterRejectsSealedDataset 断言封存数据集后，任何新机制只要作用于
// 该数据集都必须被拒绝，且不留下机制记录。
func TestRegisterRejectsSealedDataset(t *testing.T) {
	ctx := context.Background()
	svc, st := newService(t)

	// 登记并封存一个数据集
	if err := st.DatasetCreate(ctx, model.DatasetVersion{
		ID: "ds-sealed", Name: "sealed", Version: "v1",
		Status: model.DatasetSealed, EpsilonCap: 1.0, DeltaCap: 1e-5,
	}); err != nil {
		t.Fatal(err)
	}

	// 仅作用于封存数据集 → 拒绝
	if err := svc.Register(ctx, newLiveMech("m1", "ds-sealed")); !errors.Is(err, model.ErrDatasetSealed) {
		t.Fatalf("作用于封存数据集的机制应被拒绝，实际: %v", err)
	}
	if _, err := svc.Get(ctx, "m1"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("拒绝后不应留下机制记录，实际 err: %v", err)
	}

	// 多数据集且其中之一封存 → 仍拒绝（整批不落库）
	if err := st.DatasetCreate(ctx, model.DatasetVersion{
		ID: "ds-open", Name: "open", Version: "v1",
		Status: model.DatasetRegistered, EpsilonCap: 1.0, DeltaCap: 1e-5,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Register(ctx, newLiveMech("m2", "ds-open", "ds-sealed")); !errors.Is(err, model.ErrDatasetSealed) {
		t.Fatalf("含封存数据集的机制应被拒绝，实际: %v", err)
	}
	if _, err := svc.Get(ctx, "m2"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("含封存数据集拒绝后不应留下机制记录，实际 err: %v", err)
	}
}

// TestRegisterAndVerifyOnUnsealedDataset 断言未封存数据集的登记与验证正常可用。
func TestRegisterAndVerifyOnUnsealedDataset(t *testing.T) {
	ctx := context.Background()
	svc, st := newService(t)

	if err := st.DatasetCreate(ctx, model.DatasetVersion{
		ID: "ds-open", Name: "open", Version: "v1",
		Status: model.DatasetRegistered, EpsilonCap: 1.0, DeltaCap: 1e-5,
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.Register(ctx, newLiveMech("m1", "ds-open")); err != nil {
		t.Fatalf("未封存数据集登记应成功，实际: %v", err)
	}
	if err := svc.Verify(ctx, "m1"); err != nil {
		t.Fatalf("未封存数据集验证应成功，实际: %v", err)
	}
	got, err := svc.Get(ctx, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.MechVerified {
		t.Fatalf("期望 verified，实际 %s", got.Status)
	}

	// 登记后再封存数据集，原机制 Verify 不再放行（防止封存后激活为预算来源）
	if err := st.DatasetUpdate(ctx, model.DatasetVersion{
		ID: "ds-open", Name: "open", Version: "v1",
		Status: model.DatasetSealed, EpsilonCap: 1.0, DeltaCap: 1e-5,
	}); err != nil {
		t.Fatal(err)
	}
	// 回退为 draft 后再 Verify 应因数据集已封存被拒
	got.Status = model.MechDraft
	if err := st.MechanismUpdate(ctx, got); err != nil {
		t.Fatal(err)
	}
	if err := svc.Verify(ctx, "m1"); !errors.Is(err, model.ErrDatasetSealed) {
		t.Fatalf("数据集封存后 Verify 应被拒绝，实际: %v", err)
	}
}
