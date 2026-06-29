// export.go — Export (backup) profile ra ZIP bytes.
package profile

import (
	"fmt"
	"os"
	"path/filepath"

	"mingot/internal/backup"
)

// Export backup profile thành ZIP bytes để frontend trigger download.
// Lỗi nếu profile đang chạy.
func (m *Manager) Export(id string) ([]byte, error) {
	profilePath := filepath.Join(gologinDir(), id)

	if _, err := os.Stat(profilePath); err != nil {
		return nil, notFound("profile không tồn tại")
	}
	if isRunning(profilePath) {
		return nil, fmt.Errorf("chỉ có thể backup khi profile đang tắt")
	}

	return backup.Pack(profilePath)
}
