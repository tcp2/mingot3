// list.go — List tất cả profiles trong GOLOGIN_DIR.
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// List trả về danh sách tất cả profiles, sắp xếp mới nhất trước.
func (m *Manager) List() ([]Summary, error) {
	dir := gologinDir()
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		return []Summary{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var profiles []Summary
	for _, e := range entries {
		if !e.IsDir() || len(e.Name()) < 3 || e.Name()[:3] != "MG-" {
			continue
		}
		s := readSummary(e.Name())
		if !isRunning(filepath.Join(gologinDir(), e.Name())) {
			m.grid.Free(e.Name()) // auto cleanup slot nếu browser đã tắt
			s.Status = "stopped"
			s.Port = nil
		}
		profiles = append(profiles, s)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].ID > profiles[j].ID // mới nhất trước
	})
	return profiles, nil
}

func readSummary(id string) Summary {
	s := Summary{ID: id, Name: id, Status: "stopped", Folder: "Uncategorized"}
	prefsPath := filepath.Join(gologinDir(), id, "Default", "Preferences")

	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return s
	}
	var prefs struct {
		Gologin struct {
			Name   string `json:"name"`
			Port   *int   `json:"port"`
			Folder string `json:"folder"`
		} `json:"gologin"`
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return s
	}

	if prefs.Gologin.Name != "" {
		s.Name = prefs.Gologin.Name
	}
	s.Port = prefs.Gologin.Port
	if prefs.Gologin.Folder != "" {
		s.Folder = prefs.Gologin.Folder
	}
	if isRunning(filepath.Join(gologinDir(), id)) {
		s.Status = "running"
	}
	return s
}

// isRunning kiểm tra profile có đang chạy bằng cách thử rename thư mục sang chính nó.
// Trên Windows, thư mục bị lock khi Chromium đang chạy sẽ trả EBUSY/EPERM.
func isRunning(profilePath string) bool {
	if _, err := os.Stat(profilePath); err != nil {
		return false
	}
	return os.Rename(profilePath, profilePath) != nil
}
