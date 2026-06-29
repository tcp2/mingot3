// stop.go — Dừng profile browser qua CDP.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mingot/internal/cdp"
)

// Stop gửi Browser.close tới profile đang chạy.
func (m *Manager) Stop(id string) error {
	port, err := getProfilePort(id)
	if err != nil {
		return err
	}
	return cdp.CloseBrowser(port)
}

func getProfilePort(id string) (int, error) {
	prefsPath := filepath.Join(gologinDir(), id, "Default", "Preferences")
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return 0, notFound(fmt.Sprintf("profile %s không tồn tại", id))
	}

	var prefs struct {
		Gologin struct {
			Port *int `json:"port"`
		} `json:"gologin"`
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return 0, err
	}
	if prefs.Gologin.Port == nil {
		return 0, notFound("profile không có port (chưa chạy?)")
	}
	return *prefs.Gologin.Port, nil
}
