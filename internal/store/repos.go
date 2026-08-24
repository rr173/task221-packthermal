package store

import (
	"context"
	"database/sql"
)

// Repositories 聚合所有领域仓库，供 service 层注入。
type Repositories struct {
	db        *DB
	Trial     *TrialStore
	Layer     *LayerStore
	Env       *EnvStore
	Sensor    *SensorStore
	Series    *SeriesStore
	Model     *ModelStore
	Inversion *InversionStore
	Snapshot  *SnapshotStore
	Audit     *AuditStore
}

// NewRepositories 构造全部仓库。
func NewRepositories(db *DB) *Repositories {
	return &Repositories{
		db:        db,
		Trial:     &TrialStore{db: db},
		Layer:     &LayerStore{db: db},
		Env:       &EnvStore{db: db},
		Sensor:    &SensorStore{db: db},
		Series:    &SeriesStore{db: db},
		Model:     &ModelStore{db: db},
		Inversion: &InversionStore{db: db},
		Snapshot:  &SnapshotStore{db: db},
		Audit:     &AuditStore{db: db},
	}
}

// Tx 在一个事务中执行 fn；fn 返回错误则回滚。
func (r *Repositories) Tx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// CountTrials 返回试验总数（自检用）。
func (r *Repositories) CountTrials(ctx context.Context) (int, error) {
	var n int
	err := r.db.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM trials`).Scan(&n)
	return n, err
}
