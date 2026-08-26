package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"task256-privbudget/internal/model"
)

// scanDataset 从行扫描出 DatasetVersion。
func scanDataset(row interface {
	Scan(...interface{}) error
}) (model.DatasetVersion, error) {
	var d model.DatasetVersion
	var parents, sealedAt sql.NullString
	var createdAt string
	if err := row.Scan(
		&d.ID, &d.Name, &d.Version, &d.Status, &d.PrivacyUnit,
		&parents, &d.EpsilonCap, &d.DeltaCap, &createdAt, &sealedAt,
	); err != nil {
		return d, err
	}
	if err := json.Unmarshal([]byte(orDefault(parents, "[]")), &d.Parents); err != nil {
		return d, fmt.Errorf("decode parents: %w", err)
	}
	if ca, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		d.CreatedAt = ca
	}
	if sealedAt.Valid && sealedAt.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, sealedAt.String); err == nil {
			d.SealedAt = &t
		}
	}
	return d, nil
}

// DatasetCreate 插入数据集版本（幂等：已存在则报 ErrAlreadyExists）。
func (s *Store) DatasetCreate(ctx context.Context, d model.DatasetVersion) error {
	parents, err := json.Marshal(d.Parents)
	if err != nil {
		return err
	}
	now := d.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	const q = `INSERT INTO datasets (id,name,version,status,privacy_unit,parents,epsilon_cap,delta_cap,created_at,sealed_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`
	_, err = s.db.ExecContext(ctx, q,
		d.ID, d.Name, d.Version, string(d.Status), d.PrivacyUnit,
		string(parents), d.EpsilonCap, d.DeltaCap, now.Format(time.RFC3339Nano), nil,
	)
	if isUniqueViolation(err) {
		return model.ErrAlreadyExists
	}
	return err
}

// DatasetUpdate 更新数据集版本（除主键外的字段）。
func (s *Store) DatasetUpdate(ctx context.Context, d model.DatasetVersion) error {
	parents, err := json.Marshal(d.Parents)
	if err != nil {
		return err
	}
	var sealed sql.NullString
	if d.SealedAt != nil {
		sealed = sql.NullString{String: d.SealedAt.Format(time.RFC3339Nano), Valid: true}
	}
	const q = `UPDATE datasets SET name=?,version=?,status=?,privacy_unit=?,parents=?,epsilon_cap=?,delta_cap=?,sealed_at=? WHERE id=?`
	_, err = s.db.ExecContext(ctx, q,
		d.Name, d.Version, string(d.Status), d.PrivacyUnit, string(parents),
		d.EpsilonCap, d.DeltaCap, sealed, d.ID,
	)
	return err
}

// DatasetGet 按 ID 取数据集版本。
func (s *Store) DatasetGet(ctx context.Context, id string) (model.DatasetVersion, error) {
	const q = `SELECT id,name,version,status,privacy_unit,parents,epsilon_cap,delta_cap,created_at,sealed_at FROM datasets WHERE id=?`
	row := s.db.QueryRowContext(ctx, q, id)
	d, err := scanDataset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return d, model.ErrNotFound
	}
	return d, err
}

// DatasetList 列出全部数据集版本。
func (s *Store) DatasetList(ctx context.Context) ([]model.DatasetVersion, error) {
	const q = `SELECT id,name,version,status,privacy_unit,parents,epsilon_cap,delta_cap,created_at,sealed_at FROM datasets ORDER BY created_at`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DatasetVersion
	for rows.Next() {
		d, err := scanDataset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DatasetDelete 删除数据集版本（用于回滚幂等写入）。
func (s *Store) DatasetDelete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM datasets WHERE id=?`, id)
	return err
}
