package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// LayerStore 持久化箱体层序。
type LayerStore struct{ db *DB }

const layerCols = `id, trial_id, seq, material, thickness_mm, conductivity, density, specific_heat, area_m2`

func scanLayer(row interface{ Scan(...any) error }) (*model.Layer, error) {
	var l model.Layer
	if err := row.Scan(&l.ID, &l.TrialID, &l.Seq, &l.Material, &l.ThicknessMM,
		&l.Conductivity, &l.Density, &l.SpecificHeat, &l.AreaM2); err != nil {
		return nil, mapSQLError(err)
	}
	return &l, nil
}

// CreateLayer 插入一层，同试验同序号冲突返回 model.ErrDuplicate。
func (s *LayerStore) CreateLayer(ctx context.Context, l *model.Layer) (*model.Layer, error) {
	res, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO layers(trial_id, seq, material, thickness_mm, conductivity, density, specific_heat, area_m2)
		 VALUES(?,?,?,?,?,?,?,?)`,
		l.TrialID, l.Seq, l.Material, l.ThicknessMM, l.Conductivity, l.Density, l.SpecificHeat, l.AreaM2)
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
	return s.GetLayer(ctx, id)
}

// GetLayer 按 ID 读取。
func (s *LayerStore) GetLayer(ctx context.Context, id int64) (*model.Layer, error) {
	row := s.db.sql.QueryRowContext(ctx, `SELECT `+layerCols+` FROM layers WHERE id=?`, id)
	return scanLayer(row)
}

// ListLayers 列出某试验的全部层序（按 seq 升序）。
func (s *LayerStore) ListLayers(ctx context.Context, trialID int64) ([]*model.Layer, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+layerCols+` FROM layers WHERE trial_id=? ORDER BY seq`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Layer
	for rows.Next() {
		l, err := scanLayer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
