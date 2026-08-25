package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// SeriesStore 持久化温度时序。
type SeriesStore struct{ db *DB }

const seriesCols = `id, trial_id, sensor_id, fingerprint, scale, state, sample_count, samples_json, created_at`

func scanSeries(row interface{ Scan(...any) error }) (*model.TempSeries, error) {
	var s model.TempSeries
	var js, c string
	if err := row.Scan(&s.ID, &s.TrialID, &s.SensorID, &s.Fingerprint, &s.Scale,
		&s.State, &s.SampleCount, &js, &c); err != nil {
		return nil, mapSQLError(err)
	}
	s.CreatedAt, _ = parseTime(c)
	return &s, nil
}

// CreateSeries 保存一条温度时序及其采样点；同试验/传感器/指纹冲突返回 model.ErrDuplicate（幂等）。
func (s *SeriesStore) CreateSeries(ctx context.Context, srs *model.TempSeries, samples []model.TempSample) (*model.TempSeries, error) {
	js, err := marshalSamples(samples)
	if err != nil {
		return nil, err
	}
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO temp_series(trial_id, sensor_id, fingerprint, scale, state, sample_count, samples_json, created_at)
		 VALUES(?,?,?,?,?,?,?,?)`,
		srs.TrialID, srs.SensorID, srs.Fingerprint, srs.Scale, srs.State, len(samples), js, nowISO())
	if err != nil {
		if isUniqueViolation(err) {
			// 同试验/传感器/指纹的内容完全一致：保持幂等，不写入第二份，返回冲突语义。
			return nil, model.ErrDuplicate
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetSeries(ctx, id)
}

// GetSeries 按 ID 读取（含样本）。
func (s *SeriesStore) GetSeries(ctx context.Context, id int64) (*model.TempSeries, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+seriesCols+` FROM temp_series WHERE id=?`, id)
	return scanSeries(row)
}

// ListSeries 列出某试验全部温度时序。
func (s *SeriesStore) ListSeries(ctx context.Context, trialID int64) ([]*model.TempSeries, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+seriesCols+` FROM temp_series WHERE trial_id=? ORDER BY id`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.TempSeries
	for rows.Next() {
		srs, err := scanSeries(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, srs)
	}
	return out, rows.Err()
}

// ListSeriesBySensor 列出某试验某传感器的全部时序。
func (s *SeriesStore) ListSeriesBySensor(ctx context.Context, trialID, sensorID int64) ([]*model.TempSeries, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+seriesCols+` FROM temp_series WHERE trial_id=? AND sensor_id=? ORDER BY id`, trialID, sensorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.TempSeries
	for rows.Next() {
		srs, err := scanSeries(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, srs)
	}
	return out, rows.Err()
}

// UpdateSeriesState 在「当前状态 == from」前提下把时序状态改为 to。
func (s *SeriesStore) UpdateSeriesState(ctx context.Context, id int64, from, to string) error {
	res, err := s.db.sql.ExecContext(ctx,
		`UPDATE temp_series SET state=? WHERE id=? AND state=?`, to, id, from)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		cur, gerr := s.GetSeries(ctx, id)
		if gerr != nil {
			return gerr
		}
		return &model.StateError{Entity: "temp_series", ID: id, From: cur.State, To: to}
	}
	return nil
}

// LoadSamples 读取时序的采样点数组。
func (s *SeriesStore) LoadSamples(ctx context.Context, id int64) ([]model.TempSample, error) {
	var js string
	if err := s.db.sql.QueryRowContext(ctx,
		`SELECT samples_json FROM temp_series WHERE id=?`, id).Scan(&js); err != nil {
		return nil, mapSQLError(err)
	}
	return unmarshalSamples[model.TempSample](js)
}
