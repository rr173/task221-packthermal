package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// InversionStore 持久化反演结果。
type InversionStore struct{ db *DB }

const invCols = `id, trial_id, model_id, series_id, r_eff, c_eff, tau, rms_residual, converged, iterations, created_at`

func scanInversion(row interface{ Scan(...any) error }) (*model.InversionResult, error) {
	var r model.InversionResult
	var c string
	var conv int
	if err := row.Scan(&r.ID, &r.TrialID, &r.ModelID, &r.SeriesID, &r.REff, &r.CEff,
		&r.Tau, &r.RmsResidual, &conv, &r.Iterations, &c); err != nil {
		return nil, mapSQLError(err)
	}
	r.Converged = conv != 0
	r.CreatedAt, _ = parseTime(c)
	return &r, nil
}

// CreateInversion 保存一条反演结果。
func (s *InversionStore) CreateInversion(ctx context.Context, r *model.InversionResult) (*model.InversionResult, error) {
	conv := 0
	if r.Converged {
		conv = 1
	}
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO inversion_results(trial_id, model_id, series_id, r_eff, c_eff, tau, rms_residual, converged, iterations, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		r.TrialID, r.ModelID, r.SeriesID, r.REff, r.CEff, r.Tau, r.RmsResidual, conv, r.Iterations, nowISO())
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetInversion(ctx, id)
}

// GetInversion 按 ID 读取。
func (s *InversionStore) GetInversion(ctx context.Context, id int64) (*model.InversionResult, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+invCols+` FROM inversion_results WHERE id=?`, id)
	return scanInversion(row)
}

// ListInversions 列出某试验全部反演结果。
func (s *InversionStore) ListInversions(ctx context.Context, trialID int64) ([]*model.InversionResult, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+invCols+` FROM inversion_results WHERE trial_id=? ORDER BY id`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.InversionResult
	for rows.Next() {
		r, err := scanInversion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
