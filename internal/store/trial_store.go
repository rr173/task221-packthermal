package store

import (
	"context"
	"strings"

	"task221-packthermal/internal/model"
)

// TrialStore 持久化验证试验。
type TrialStore struct{ db *DB }

const trialCols = `id, code, title, state, created_at, updated_at`

func scanTrial(row interface{ Scan(...any) error }) (*model.Trial, error) {
	var t model.Trial
	var c, u string
	if err := row.Scan(&t.ID, &t.Code, &t.Title, &t.State, &c, &u); err != nil {
		return nil, mapSQLError(err)
	}
	t.CreatedAt, _ = parseTime(c)
	t.UpdatedAt, _ = parseTime(u)
	return &t, nil
}

// CreateTrial 新建试验，code 冲突返回 model.ErrDuplicate。
func (s *TrialStore) CreateTrial(ctx context.Context, code, title string) (*model.Trial, error) {
	now := nowISO()
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO trials(code, title, state, created_at, updated_at) VALUES(?,?,?,?,?)`,
		code, title, model.TrialPlanned, now, now)
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
	return s.GetTrial(ctx, id)
}

// GetTrial 按 ID 读取。
func (s *TrialStore) GetTrial(ctx context.Context, id int64) (*model.Trial, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+trialCols+` FROM trials WHERE id=?`, id)
	return scanTrial(row)
}

// GetTrialByCode 按编号读取。
func (s *TrialStore) GetTrialByCode(ctx context.Context, code string) (*model.Trial, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+trialCols+` FROM trials WHERE code=?`, code)
	return scanTrial(row)
}

// ListTrials 列出全部试验。
func (s *TrialStore) ListTrials(ctx context.Context) ([]*model.Trial, error) {
	rows, err := s.db.sql.QueryContext(ctx, `SELECT `+trialCols+` FROM trials ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Trial
	for rows.Next() {
		t, err := scanTrial(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TransitionState 在「当前状态 == from」的前提下把状态改为 to，实现原子状态机流转。
func (s *TrialStore) TransitionState(ctx context.Context, id int64, from, to string) error {
	res, err := s.db.sql.ExecContext(ctx,
		`UPDATE trials SET state=?, updated_at=? WHERE id=? AND state=?`,
		to, nowISO(), id, from)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		cur, gerr := s.GetTrial(ctx, id)
		if gerr != nil {
			return gerr
		}
		return &model.StateError{Entity: "trial", ID: id, From: cur.State, To: to}
	}
	return nil
}

// isUniqueViolation 判断是否唯一约束冲突。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}
