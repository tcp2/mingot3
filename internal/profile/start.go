// start.go — Khởi chạy profile browser và vị trí cửa sổ.
package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"mingot/internal/backup"
	"mingot/internal/cdp"
	"mingot/internal/orbita"
)

// Start khởi chạy profile, trả về debug port.
func (m *Manager) Start(id string, screenW, screenH int) (*StartResult, error) {
	profilePath := filepath.Join(gologinDir(), id)
	if _, err := os.Stat(profilePath); err != nil {
		return nil, notFound("profile không tồn tại")
	}

	slotIndex := m.grid.Allocate(id)
	gridCfg := m.grid.Config(slotIndex, screenW, screenH)

	l, err := orbita.New(id)
	if err != nil {
		m.grid.Free(id)
		return nil, err
	}
	if err := l.Init(); err != nil {
		m.grid.Free(id)
		return nil, err
	}
	if err := l.BuildProfile(); err != nil {
		m.grid.Free(id)
		return nil, err
	}

	onClose := m.makeOnClose(id, profilePath, l.DebugPort)
	if err := l.Launch(&gridCfg, onClose); err != nil {
		m.grid.Free(id)
		return nil, err
	}

	// Vị trí cửa sổ đã được set trước bởi Launch, nhưng đảm bảo thêm lần nữa
	go cdp.SetWindowBounds(l.DebugPort, gridCfg.X, gridCfg.Y, gridCfg.W, gridCfg.H) //nolint:errcheck

	return &StartResult{Port: l.DebugPort}, nil
}

// makeOnClose tạo closure xử lý dọn dẹp sau khi browser đóng.
func (m *Manager) makeOnClose(id, profilePath string, _ int) func() {
	return func() {
		fmt.Printf("[PROFILE] Browser %s đã đóng\n", id)
		m.grid.Free(id)
		m.emit("profile:closed", id)

		go func() {
			m.emit("profile:backing-up", id)
			clearPort(profilePath)
			backup.Sanitize(profilePath)
			m.emit("profile:backed-up", id)
		}()
	}
}

func clearPort(profilePath string) {
	prefsPath := filepath.Join(profilePath, "Default", "Preferences")
	prefs, err := readPrefsMap(prefsPath)
	if err != nil {
		return
	}
	gl, ok := prefs["gologin"].(map[string]any)
	if !ok {
		return
	}
	gl["port"] = nil
	_ = writePrefsMap(prefsPath, prefs)
}
