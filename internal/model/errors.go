package model

import (
	"errors"
	"fmt"
)

// 领域错误，用于 service 层到 HTTP 层的错误映射。
var (
	ErrNotFound      = errors.New("entity not found")
	ErrConflict      = errors.New("state conflict")
	ErrDuplicate     = errors.New("duplicate entity")
	ErrInvalidInput  = errors.New("invalid input")
	ErrImmutable     = errors.New("immutable entity")
)

// StateError 表示一次非法状态机流转。
type StateError struct {
	Entity string
	ID     int64
	From   string
	To     string
}

func (e *StateError) Error() string {
	return fmt.Sprintf("illegal state transition for %s#%d: %s -> %s", e.Entity, e.ID, e.From, e.To)
}

// Wrap 返回一个携带上下文信息的错误。
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
