// Package profile cung cấp toàn bộ CRUD operations cho browser profiles.
package profile

import "errors"

// ─── Domain types ─────────────────────────────────────────────────────────────

// Summary là thông tin hiển thị của một profile.
type Summary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Port   *int   `json:"port"`
	Status string `json:"status"` // "running" | "stopped"
	Folder string `json:"folder"`
}

// StartResult trả về sau khi start profile thành công.
type StartResult struct {
	Port int `json:"port"`
}

// CreateResult trả về sau khi tạo profile mới.
type CreateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ImportResult trả về sau khi import profile từ ZIP.
type ImportResult struct {
	ID string `json:"id"`
}

// ─── Error types ──────────────────────────────────────────────────────────────

// ErrCode phân loại lỗi để frontend xử lý đúng.
type ErrCode string

const (
	CodeNotFound ErrCode = "NOT_FOUND"
	CodeRunning  ErrCode = "RUNNING"
	CodeConflict ErrCode = "CONFLICT"
)

// AppError là lỗi có code để frontend phân biệt.
type AppError struct {
	Code      ErrCode
	Message   string
	ProfileID string // chỉ dùng cho CONFLICT
}

func (e *AppError) Error() string { return e.Message }

func notFound(msg string) *AppError       { return &AppError{Code: CodeNotFound, Message: msg} }
func profileRunning(msg string) *AppError { return &AppError{Code: CodeRunning, Message: msg} }
func conflict(id, msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg, ProfileID: id}
}

// IsNotFound trả về true nếu err là AppError với code NOT_FOUND.
func IsNotFound(err error) bool {
	var ae *AppError
	return errors.As(err, &ae) && ae.Code == CodeNotFound
}
