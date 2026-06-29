// prefs_io.go — Helpers đọc/ghi Preferences file dùng chung trong package profile.
package profile

import (
	"encoding/json"
	"os"
)

func readPrefsMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	return m, json.Unmarshal(data, &m)
}

func writePrefsMap(path string, v map[string]any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
