package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// SnapshotStore 持久化验证快照（版本化、不可变）。
type SnapshotStore struct{ db *DB }

const snapCols = `id, trial_id, version, state, summary, created_at`

func scanSnapshot(row interface{ Scan(...any) error }) (*model.Snapshot, error) {
	var s model.Snapshot
	var c string
	if err := row.Scan(&s.ID, &s.TrialID, &s.Version, &s.State, &s.Summary, &c); err != nil {
		return nil, mapSQLError(err)
	}
	s.CreatedAt, _ = parseTime(c)
	return &s, nil
}

// NextVersion 返回某试验下一个可用版本号。
func (s *SnapshotStore) NextVersion(ctx context.Context, trialID int64) (int, error) {
	var v int
	err := s.db.sql.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM snapshots WHERE trial_id=?`, trialID).Scan(&v)
	if err != nil {
		return 0, err
	}
	return v + 1, nil
}

// CreateSnapshot 发布一条快照，同试验同版本冲突返回 model.ErrDuplicate。
func (s *SnapshotStore) CreateSnapshot(ctx context.Context, sn *model.Snapshot) (*model.Snapshot, error) {
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO snapshots(trial_id, version, state, summary, created_at) VALUES(?,?,?,?,?)`,
		sn.TrialID, sn.Version, sn.State, sn.Summary, nowISO())
	if err != nil {
		if isUniqueViolation(err) {
			return nil, model.ErrDuplicate
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetSnapshot(ctx, id)
}

// GetSnapshot 按 ID 读取。
func (s *SnapshotStore) GetSnapshot(ctx context.Context, id int64) (*model.Snapshot, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+snapCols+` FROM snapshots WHERE id=?`, id)
	return scanSnapshot(row)
}

// ListSnapshots 列出某试验全部快照（按版本升序）。
func (s *SnapshotStore) ListSnapshots(ctx context.Context, trialID int64) ([]*model.Snapshot, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+snapCols+` FROM snapshots WHERE trial_id=? ORDER BY version`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		sn, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

// UpdateSnapshotState 在「当前状态 == from」前提下把快照状态改为 to。
func (s *SnapshotStore) UpdateSnapshotState(ctx context.Context, id int64, from, to string) error {
	res, err := s.db.sql.ExecContext(ctx,
		`UPDATE snapshots SET state=? WHERE id=? AND state=?`, to, id, from)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		cur, gerr := s.GetSnapshot(ctx, id)
		if gerr != nil {
			return gerr
		}
		return &model.StateError{Entity: "snapshot", ID: id, From: cur.State, To: to}
	}
	return nil
}

// SupersedePublished 把某试验已发布的快照降级为「替代」。
func (s *SnapshotStore) SupersedePublished(ctx context.Context, trialID int64) (int, error) {
	res, err := s.db.sql.ExecContext(ctx,
		`UPDATE snapshots SET state=? WHERE trial_id=? AND state=?`,
		model.SnapshotSuperseded, trialID, model.SnapshotPublished)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}
