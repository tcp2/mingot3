// folder.go — Cập nhật folder cho danh sách profiles.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
)

// UpdateFolder gán folder mới cho nhiều profiles cùng lúc.
func (m *Manager) UpdateFolder(ids []string, folder string) error {
	for _, id := range ids {
		if err := setFolder(id, folder); err != nil {
			return fmt.Errorf("profile %s: %w", id, err)
		}
	}
	return nil
}

func setFolder(id, folder string) error {
	prefsPath := filepath.Join(gologinDir(), id, "Default", "Preferences")
	if _, err := os.Stat(prefsPath); err != nil {
		return nil // bỏ qua profile không có Preferences
	}

	prefs, err := readPrefsMap(prefsPath)
	if err != nil {
		return err
	}
	if prefs["gologin"] == nil {
		prefs["gologin"] = map[string]any{}
	}
	prefs["gologin"].(map[string]any)["folder"] = folder
	return writePrefsMap(prefsPath, prefs)
}
