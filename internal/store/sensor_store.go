package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// SensorStore 持久化温度传感器。
type SensorStore struct{ db *DB }

const sensorCols = `id, trial_id, code, position, scale, created_at`

func scanSensor(row interface{ Scan(...any) error }) (*model.Sensor, error) {
	var s model.Sensor
	var c string
	if err := row.Scan(&s.ID, &s.TrialID, &s.Code, &s.Position, &s.Scale, &c); err != nil {
		return nil, mapSQLError(err)
	}
	s.CreatedAt, _ = parseTime(c)
	return &s, nil
}

// CreateSensor 插入一只传感器，同试验同编号冲突返回 model.ErrDuplicate。
func (s *SensorStore) CreateSensor(ctx context.Context, sn *model.Sensor) (*model.Sensor, error) {
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO sensors(trial_id, code, position, scale, created_at) VALUES(?,?,?,?,?)`,
		sn.TrialID, sn.Code, sn.Position, sn.Scale, nowISO())
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
	return s.GetSensor(ctx, id)
}

// GetSensor 按 ID 读取。
func (s *SensorStore) GetSensor(ctx context.Context, id int64) (*model.Sensor, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+sensorCols+` FROM sensors WHERE id=?`, id)
	return scanSensor(row)
}

// ListSensors 列出某试验的全部传感器。
func (s *SensorStore) ListSensors(ctx context.Context, trialID int64) ([]*model.Sensor, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+sensorCols+` FROM sensors WHERE trial_id=? ORDER BY id`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Sensor
	for rows.Next() {
		sn, err := scanSensor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}
