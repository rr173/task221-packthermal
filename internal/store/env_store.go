package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// EnvStore 持久化环境温度阶跃曲线。
type EnvStore struct{ db *DB }

const envCols = `id, trial_id, name, fingerprint, samples_json, created_at`

func scanEnv(row interface{ Scan(...any) error }) (*model.EnvProfile, error) {
	var e model.EnvProfile
	var js, c string
	if err := row.Scan(&e.ID, &e.TrialID, &e.Name, &e.Fingerprint, &js, &c); err != nil {
		return nil, mapSQLError(err)
	}
	samples, err := unmarshalSamples[model.EnvSample](js)
	if err != nil {
		return nil, err
	}
	e.Samples = samples
	e.CreatedAt, _ = parseTime(c)
	return &e, nil
}

// CreateEnv 保存环境曲线，同试验同指纹幂等（冲突返回 model.ErrDuplicate）。
func (s *EnvStore) CreateEnv(ctx context.Context, e *model.EnvProfile) (*model.EnvProfile, error) {
	js, err := marshalSamples(e.Samples)
	if err != nil {
		return nil, err
	}
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO env_profiles(trial_id, name, fingerprint, samples_json, created_at) VALUES(?,?,?,?,?)`,
		e.TrialID, e.Name, e.Fingerprint, js, nowISO())
	if err != nil {
		if isUniqueViolation(err) {
			return nil, model.ErrInvalidInput
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetEnv(ctx, id)
}

// GetEnv 按 ID 读取。
func (s *EnvStore) GetEnv(ctx context.Context, id int64) (*model.EnvProfile, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+envCols+` FROM env_profiles WHERE id=?`, id)
	return scanEnv(row)
}

// ListEnvs 列出某试验的全部环境曲线。
func (s *EnvStore) ListEnvs(ctx context.Context, trialID int64) ([]*model.EnvProfile, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+envCols+` FROM env_profiles WHERE trial_id=? ORDER BY id`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.EnvProfile
	for rows.Next() {
		e, err := scanEnv(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
