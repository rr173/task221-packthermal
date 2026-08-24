package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"task221-packthermal/internal/model"
)

// timeLayout 是所有时间戳的统一存储格式（UTC，秒级精度）。
const timeLayout = "2006-01-02T15:04:05Z"

func nowISO() string { return time.Now().UTC().Format(timeLayout) }

func parseTime(s string) (time.Time, error) { return time.Parse(timeLayout, s) }

// mapSQLError 把驱动层错误映射为领域错误。
func mapSQLError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrNotFound
	}
	return err
}

// marshalSamples 把采样点数组序列化为 JSON 文本。
func marshalSamples[T any](samples []T) (string, error) {
	b, err := json.Marshal(samples)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalSamples 把 JSON 文本反序列化为采样点数组。
func unmarshalSamples[T any](text string) ([]T, error) {
	var out []T
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, err
	}
	return out, nil
}
