// manager.go — Manager là entry point cho tất cả profile operations.
package profile

import (
	"context"

	"mingot/internal/grid"
	"mingot/internal/storage"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
)

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
