package store

import (
	"context"

	"task221-packthermal/internal/model"
)

// DefaultAuditLimit 是未指定 limit 时返回的默认事件条数。
const DefaultAuditLimit = 100

// AuditStore 持久化审计台账。
type AuditStore struct{ db *DB }

const auditCols = `id, trial_id, action, detail, created_at`

func scanAudit(row interface{ Scan(...any) error }) (*model.AuditEvent, error) {
	var a model.AuditEvent
	var c string
	if err := row.Scan(&a.ID, &a.TrialID, &a.Action, &a.Detail, &c); err != nil {
		return nil, mapSQLError(err)
	}
	a.CreatedAt, _ = parseTime(c)
	return &a, nil
}

// Record 记录一条审计事件。
func (s *AuditStore) Record(ctx context.Context, trialID int64, action, detail string) error {
	_, err := s.db.sql.ExecContext(ctx,
		`INSERT INTO audit_events(trial_id, action, detail, created_at) VALUES(?,?,?,?)`,
		trialID, action, detail, nowISO())
	return err
}

// List 列出某试验的审计事件。
func (s *AuditStore) List(ctx context.Context, trialID int64) ([]*model.AuditEvent, error) {
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+auditCols+` FROM audit_events WHERE trial_id=? ORDER BY id`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.AuditEvent
	for rows.Next() {
		a, err := scanAudit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListAll 列出全部审计事件。limit<=0 时按默认条数返回，避免零值层层传递导致空结果。
func (s *AuditStore) ListAll(ctx context.Context, limit int) ([]*model.AuditEvent, error) {
	if limit <= 0 {
		limit = DefaultAuditLimit
	}
	rows, err := s.db.sql.QueryContext(ctx,
		`SELECT `+auditCols+` FROM audit_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.AuditEvent
	for rows.Next() {
		a, err := scanAudit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
