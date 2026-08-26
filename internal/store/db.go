// Package store 使用纯 Go 的 modernc.org/sqlite 驱动持久化领域实体。
// 所有写操作在应用层被串行化（见 service 包锁），避免 SQLite 写锁竞争；
// 读操作可并发。数据库关闭重开后仍能恢复未完成的评估与已冻结快照。
package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Store 封装 *sql.DB 与建表迁移。
type Store struct {
	db *sql.DB
}

// Open 打开（或创建）SQLite 数据库并完成迁移。
func Open(path string) (*Store, error) {
	dsn := "file:" + path +
		"?_pragma=busy_timeout(8000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(0)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭底层连接。
func (s *Store) Close() error { return s.db.Close() }

// DB 暴露底层连接（供事务使用）。
func (s *Store) DB() *sql.DB { return s.db }

// migrate 建立全部表（幂等）。
func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS datasets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			status TEXT NOT NULL,
			privacy_unit TEXT NOT NULL DEFAULT '',
			parents TEXT NOT NULL DEFAULT '[]',
			epsilon_cap REAL NOT NULL DEFAULT 0,
			delta_cap REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			sealed_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS mechanisms (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			epsilon REAL NOT NULL DEFAULT 0,
			delta REAL NOT NULL DEFAULT 0,
			dataset_ids TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL,
			validated_at TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS releases (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			mechanism_id TEXT NOT NULL,
			rule TEXT NOT NULL,
			status TEXT NOT NULL,
			evaluated_at TEXT,
			overlimit_path TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			rule TEXT NOT NULL,
			rule_version TEXT NOT NULL,
			status TEXT NOT NULL,
			entries TEXT NOT NULL DEFAULT '[]',
			summary TEXT NOT NULL DEFAULT '',
			frozen_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_releases_mechanism ON releases(mechanism_id)`,
		`CREATE INDEX IF NOT EXISTS idx_releases_status ON releases(status)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return err
		}
	}
	return nil
}
