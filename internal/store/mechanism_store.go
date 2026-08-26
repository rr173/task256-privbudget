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

func scanMechanism(row interface {
	Scan(...interface{}) error
}) (model.Mechanism, error) {
	var m model.Mechanism
	var datasets, validatedAt sql.NullString
	var createdAt string
	if err := row.Scan(
		&m.ID, &m.Name, &m.Kind, &m.Epsilon, &m.Delta,
		&datasets, &m.Status, &validatedAt, &createdAt,
	); err != nil {
		return m, err
	}
	if err := json.Unmarshal([]byte(orDefault(datasets, "[]")), &m.DatasetIDs); err != nil {
		return m, fmt.Errorf("decode dataset_ids: %w", err)
	}
	if ca, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		m.CreatedAt = ca
	}
	if validatedAt.Valid && validatedAt.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, validatedAt.String); err == nil {
			m.ValidatedAt = &t
		}
	}
	return m, nil
}

// MechanismCreate 插入统计机制（幂等）。
func (s *Store) MechanismCreate(ctx context.Context, m model.Mechanism) error {
	datasets, err := json.Marshal(m.DatasetIDs)
	if err != nil {
		return err
	}
	now := m.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	const q = `INSERT INTO mechanisms (id,name,kind,epsilon,delta,dataset_ids,status,validated_at,created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`
	_, err = s.db.ExecContext(ctx, q,
		m.ID, m.Name, string(m.Kind), m.Epsilon, m.Delta, string(datasets),
		string(m.Status), nil, now.Format(time.RFC3339Nano),
	)
	if isUniqueViolation(err) {
		return model.ErrAlreadyExists
	}
	return err
}

// MechanismUpdate 更新机制（除主键外）。
func (s *Store) MechanismUpdate(ctx context.Context, m model.Mechanism) error {
	datasets, err := json.Marshal(m.DatasetIDs)
	if err != nil {
		return err
	}
	var validated sql.NullString
	if m.ValidatedAt != nil {
		validated = sql.NullString{String: m.ValidatedAt.Format(time.RFC3339Nano), Valid: true}
	}
	const q = `UPDATE mechanisms SET name=?,kind=?,epsilon=?,delta=?,dataset_ids=?,status=?,validated_at=? WHERE id=?`
	_, err = s.db.ExecContext(ctx, q,
		m.Name, string(m.Kind), m.Epsilon, m.Delta, string(datasets), string(m.Status), validated, m.ID,
	)
	return err
}

// MechanismGet 按 ID 取机制。
func (s *Store) MechanismGet(ctx context.Context, id string) (model.Mechanism, error) {
	const q = `SELECT id,name,kind,epsilon,delta,dataset_ids,status,validated_at,created_at FROM mechanisms WHERE id=?`
	row := s.db.QueryRowContext(ctx, q, id)
	m, err := scanMechanism(row)
	if errors.Is(err, sql.ErrNoRows) {
		return m, model.ErrNotFound
	}
	return m, err
}

// MechanismList 列出全部机制。
func (s *Store) MechanismList(ctx context.Context) ([]model.Mechanism, error) {
	const q = `SELECT id,name,kind,epsilon,delta,dataset_ids,status,validated_at,created_at FROM mechanisms ORDER BY created_at`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Mechanism
	for rows.Next() {
		m, err := scanMechanism(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MechanismDelete 删除机制（回滚幂等写入用）。
func (s *Store) MechanismDelete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mechanisms WHERE id=?`, id)
	return err
}
