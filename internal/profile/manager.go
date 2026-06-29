package profile

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"mingot/internal/grid"
	"mingot/internal/storage"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ─── Manager ────────────────--------------------------------------------------

// Manager tổng hợp các dependencies cần thiết cho profile operations.
type Manager struct {
	grid *grid.Tracker
	ctx  context.Context
}

// New tạo Manager mới với grid tracker riêng.
func New() *Manager {
	return &Manager{grid: grid.New()}
}

// SetContext được gọi từ app.go sau khi Wails cấp context (OnStartup).
func (m *Manager) SetContext(ctx context.Context) { m.ctx = ctx }

// emit phát event lên frontend qua Wails runtime.
func (m *Manager) emit(eventType, profileID string) {
	if m.ctx == nil {
		return
	}
	wailsrt.EventsEmit(m.ctx, "profile:event", map[string]string{
		"type": eventType,
		"id":   profileID,
	})
}

// GologinDir trả về đường dẫn thư mục profiles (helper dùng nội bộ).
func gologinDir() string { return storage.GologinDir() }

// ─── Domain models ────────────────--------------------------------------------

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

// ─── Error handling ────────────────-------------------------------------------

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

// ─── Preferences IO Helpers ────────────────------------------------------------

func readPrefsMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	return out, json.Unmarshal(data, &out)
}

func writePrefsMap(path string, m map[string]any) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
