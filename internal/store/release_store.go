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

func scanRelease(row interface {
	Scan(...interface{}) error
}) (model.Release, error) {
	var r model.Release
	var evaluatedAt, path sql.NullString
	var createdAt string
	if err := row.Scan(
		&r.ID, &r.Name, &r.MechanismID, &r.Rule, &r.Status, &evaluatedAt, &path, &createdAt,
	); err != nil {
		return r, err
	}
	if err := json.Unmarshal([]byte(orDefault(path, "[]")), &r.OverlimitPath); err != nil {
		return r, fmt.Errorf("decode overlimit_path: %w", err)
	}
	if ca, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		r.CreatedAt = ca
	}
	if evaluatedAt.Valid && evaluatedAt.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, evaluatedAt.String); err == nil {
			r.EvaluatedAt = &t
		}
	}
	return r, nil
}

// ReleaseCreate 插入发布批次（幂等）。
func (s *Store) ReleaseCreate(ctx context.Context, r model.Release) error {
	path, err := json.Marshal(r.OverlimitPath)
	if err != nil {
		return err
	}
	now := r.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	const q = `INSERT INTO releases (id,name,mechanism_id,rule,status,evaluated_at,overlimit_path,created_at)
		VALUES (?,?,?,?,?,?,?,?)`
	_, err = s.db.ExecContext(ctx, q,
		r.ID, r.Name, r.MechanismID, string(r.Rule), string(r.Status), nil, string(path),
		now.Format(time.RFC3339Nano),
	)
	if isUniqueViolation(err) {
		return model.ErrAlreadyExists
	}
	return err
}

// ReleaseUpdate 更新发布（除主键外）。
func (s *Store) ReleaseUpdate(ctx context.Context, r model.Release) error {
	path, err := json.Marshal(r.OverlimitPath)
	if err != nil {
		return err
	}
	var evaluated sql.NullString
	if r.EvaluatedAt != nil {
		evaluated = sql.NullString{String: r.EvaluatedAt.Format(time.RFC3339Nano), Valid: true}
	}
	const q = `UPDATE releases SET name=?,mechanism_id=?,rule=?,status=?,evaluated_at=?,overlimit_path=? WHERE id=?`
	_, err = s.db.ExecContext(ctx, q,
		r.Name, r.MechanismID, string(r.Rule), string(r.Status), evaluated, string(path), r.ID,
	)
	return err
}

// ReleaseGet 按 ID 取发布。
func (s *Store) ReleaseGet(ctx context.Context, id string) (model.Release, error) {
	const q = `SELECT id,name,mechanism_id,rule,status,evaluated_at,overlimit_path,created_at FROM releases WHERE id=?`
	row := s.db.QueryRowContext(ctx, q, id)
	r, err := scanRelease(row)
	if errors.Is(err, sql.ErrNoRows) {
		return r, model.ErrNotFound
	}
	return r, err
}

// ReleaseList 列出全部发布。
func (s *Store) ReleaseList(ctx context.Context) ([]model.Release, error) {
	const q = `SELECT id,name,mechanism_id,rule,status,evaluated_at,overlimit_path,created_at FROM releases ORDER BY created_at`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Release
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReleaseDelete 删除发布。
func (s *Store) ReleaseDelete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM releases WHERE id=?`, id)
	return err
}
