package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// ModelStore 持久化热网络模型。
type ModelStore struct{ db *DB }

const modelCols = `id, trial_id, name, state, notes, created_at`

func scanModel(row interface{ Scan(...any) error }) (*model.ThermalModel, error) {
	var m model.ThermalModel
	var c string
	if err := row.Scan(&m.ID, &m.TrialID, &m.Name, &m.State, &m.Notes, &c); err != nil {
		return nil, mapSQLError(err)
	}
	m.CreatedAt, _ = parseTime(c)
	return &m, nil
}

// CreateModel 新建热模型。
func (s *ModelStore) CreateModel(ctx context.Context, m *model.ThermalModel) (*model.ThermalModel, error) {
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO thermal_models(trial_id, name, state, notes, created_at) VALUES(?,?,?,?,?)`,
		m.TrialID, m.Name, m.State, m.Notes, nowISO())
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetModel(ctx, id)
}

// GetModel 按 ID 读取。
func (s *ModelStore) GetModel(ctx context.Context, id int64) (*model.ThermalModel, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+modelCols+` FROM thermal_models WHERE id=?`, id)
	return scanModel(row)
}

// ListModels 列出某试验全部模型。
func (s *ModelStore) ListModels(ctx context.Context, trialID int64) ([]*model.ThermalModel, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+modelCols+` FROM thermal_models WHERE trial_id=? ORDER BY id`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ThermalModel
	for rows.Next() {
		m, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateModelState 在「当前状态 == from」前提下把模型状态改为 to。
func (s *ModelStore) UpdateModelState(ctx context.Context, id int64, from, to string) error {
	res, err := s.db.sql.ExecContext(ctx,
		`UPDATE thermal_models SET state=? WHERE id=? AND state=?`, to, id, from)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		cur, gerr := s.GetModel(ctx, id)
		if gerr != nil {
			return gerr
		}
		return &model.StateError{Entity: "thermal_model", ID: id, From: cur.State, To: to}
	}
	return nil
}
