// Package store 提供基于 SQLite 的持久化层。
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB 是持久化句柄，包一层 *sql.DB。
type DB struct {
	sql *sql.DB
}

// Open 打开（必要时创建）指定路径的 SQLite 数据库并执行建表迁移。
func Open(path string) (*DB, error) {
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// 单写入者 + WAL 兼顾并发读；busy_timeout 避免多连接写入竞争直接报错。
	if _, err := raw.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); err != nil {
		raw.Close()
		return nil, fmt.Errorf("configure sqlite: %w", err)
	}
	db := &DB{sql: raw}
	if err := db.migrate(); err != nil {
		raw.Close()
		return nil, err
	}
	return db, nil
}

// Close 关闭数据库连接。
func (db *DB) Close() error { return db.sql.Close() }

// migrate 创建全部业务表与唯一索引。
func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL DEFAULT '',
			state TEXT NOT NULL DEFAULT 'planned',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS layers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			seq INTEGER NOT NULL,
			material TEXT NOT NULL,
			thickness_mm REAL NOT NULL,
			conductivity REAL NOT NULL,
			density REAL NOT NULL,
			specific_heat REAL NOT NULL,
			area_m2 REAL NOT NULL,
			UNIQUE(trial_id, seq)
		);`,
		`CREATE TABLE IF NOT EXISTS env_profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			fingerprint TEXT NOT NULL,
			samples_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, fingerprint)
		);`,
		`CREATE TABLE IF NOT EXISTS sensors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			code TEXT NOT NULL,
			position TEXT NOT NULL,
			scale TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, code)
		);`,
		`CREATE TABLE IF NOT EXISTS temp_series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			sensor_id INTEGER NOT NULL,
			fingerprint TEXT NOT NULL,
			scale TEXT NOT NULL,
			state TEXT NOT NULL DEFAULT 'pending',
			sample_count INTEGER NOT NULL,
			samples_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, sensor_id, fingerprint)
		);`,
		`CREATE TABLE IF NOT EXISTS thermal_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			state TEXT NOT NULL DEFAULT 'draft',
			notes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS inversion_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			model_id INTEGER NOT NULL,
			series_id INTEGER NOT NULL,
			r_eff REAL NOT NULL,
			c_eff REAL NOT NULL,
			tau REAL NOT NULL,
			rms_residual REAL NOT NULL,
			converged INTEGER NOT NULL,
			iterations INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			version INTEGER NOT NULL,
			state TEXT NOT NULL DEFAULT 'computing',
			summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, version)
		);`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_layers_trial ON layers(trial_id);`,
		`CREATE INDEX IF NOT EXISTS idx_series_trial ON temp_series(trial_id);`,
		`CREATE INDEX IF NOT EXISTS idx_inversions_trial ON inversion_results(trial_id);`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_trial ON snapshots(trial_id);`,
	}
	for _, s := range stmts {
		if _, err := db.sql.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
