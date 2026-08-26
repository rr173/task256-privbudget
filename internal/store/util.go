package store

import (
	"database/sql"
	"strings"

	"task256-privbudget/internal/model"
)

// orDefault 当字符串为空时返回默认值，主要用于 TEXT 列的 JSON 兜底。
func orDefault(s sql.NullString, def string) string {
	if s.Valid && s.String != "" {
		return s.String
	}
	return def
}

// isUniqueViolation 判断驱动错误是否为唯一约束冲突（sqlite 错误码 2067/1555）。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "2067") ||
		strings.Contains(msg, "1555")
}

// isNotFound 判断是否为“无此行”错误。
func isNotFound(err error) bool {
	return err != nil && err.Error() == model.ErrNotFound.Error()
}
