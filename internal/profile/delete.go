// delete.go — Xoá profile khỏi disk.
package profile

import (
	"os"
	"path/filepath"
)

// Delete xoá thư mục profile. Lỗi nếu profile đang chạy hoặc không tồn tại.
func (m *Manager) Delete(id string) error {
	profilePath := filepath.Join(gologinDir(), id)

	if _, err := os.Stat(profilePath); err != nil {
		return notFound("profile không tồn tại")
	}
	if isRunning(profilePath) {
		return profileRunning("không thể xoá profile đang chạy, hãy dừng trước")
	}

	if err := os.RemoveAll(profilePath); err != nil {
		return err
	}

	// Xoá file .zip backup nếu có
	zipPath := filepath.Join(gologinDir(), id+".zip")
	_ = os.Remove(zipPath) // bỏ qua lỗi nếu không có

	return nil
}
