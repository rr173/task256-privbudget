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

func scanSnapshot(row interface {
	Scan(...interface{}) error
}) (model.BudgetSnapshot, error) {
	var s model.BudgetSnapshot
	var entries, frozenAt sql.NullString
	var createdAt string
	if err := row.Scan(
		&s.ID, &s.Name, &s.Rule, &s.RuleVersion, &s.Status, &entries, &s.Summary, &frozenAt, &createdAt,
	); err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(orDefault(entries, "[]")), &s.Entries); err != nil {
		return s, fmt.Errorf("decode entries: %w", err)
	}
	if ca, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		s.CreatedAt = ca
	}
	if frozenAt.Valid && frozenAt.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, frozenAt.String); err == nil {
			s.FrozenAt = t
		}
	}
	return s, nil
}

// SnapshotCreate 插入预算快照（幂等）。
func (s *Store) SnapshotCreate(ctx context.Context, snap model.BudgetSnapshot) error {
	entries, err := json.Marshal(snap.Entries)
	if err != nil {
		return err
	}
	now := snap.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	frozen := snap.FrozenAt
	if frozen.IsZero() {
		frozen = now
	}
	const q = `INSERT INTO snapshots (id,name,rule,rule_version,status,entries,summary,frozen_at,created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`
	_, err = s.db.ExecContext(ctx, q,
		snap.ID, snap.Name, string(snap.Rule), snap.RuleVersion, string(snap.Status),
		string(entries), snap.Summary, frozen.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano),
	)
	if isUniqueViolation(err) {
		return model.ErrAlreadyExists
	}
	return err
}

// SnapshotUpdate 更新快照（除主键外）。
func (s *Store) SnapshotUpdate(ctx context.Context, snap model.BudgetSnapshot) error {
	entries, err := json.Marshal(snap.Entries)
	if err != nil {
		return err
	}
	const q = `UPDATE snapshots SET name=?,rule=?,rule_version=?,status=?,entries=?,summary=?,frozen_at=? WHERE id=?`
	_, err = s.db.ExecContext(ctx, q,
		snap.Name, string(snap.Rule), snap.RuleVersion, string(snap.Status), string(entries),
		snap.Summary, snap.FrozenAt.Format(time.RFC3339Nano), snap.ID,
	)
	return err
}

// SnapshotGet 按 ID 取快照。
func (s *Store) SnapshotGet(ctx context.Context, id string) (model.BudgetSnapshot, error) {
	const q = `SELECT id,name,rule,rule_version,status,entries,summary,frozen_at,created_at FROM snapshots WHERE id=?`
	row := s.db.QueryRowContext(ctx, q, id)
	snap, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return snap, model.ErrNotFound
	}
	return snap, err
}

// SnapshotList 列出全部快照。
func (s *Store) SnapshotList(ctx context.Context) ([]model.BudgetSnapshot, error) {
	const q = `SELECT id,name,rule,rule_version,status,entries,summary,frozen_at,created_at FROM snapshots ORDER BY created_at`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.BudgetSnapshot
	for rows.Next() {
		snap, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

// SnapshotDelete 删除快照。
func (s *Store) SnapshotDelete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM snapshots WHERE id=?`, id)
	return err
}
