// import.go — Import profile từ ZIP bytes.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mingot/internal/util"
	pkgzip "mingot/internal/zip"
)

// Import giải nén ZIP, phát hiện profileID, copy vào GOLOGIN_DIR.
// overwrite = true cho phép ghi đè nếu profile đã tồn tại.
func (m *Manager) Import(zipData []byte, overwrite bool) (*ImportResult, error) {
	tmpDir, err := os.MkdirTemp("", "mg-import-*")
	if err != nil {
		return nil, fmt.Errorf("tạo thư mục tạm: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := pkgzip.Extract(zipData, tmpDir); err != nil {
		return nil, fmt.Errorf("giải nén thất bại: %w", err)
	}

	finalID, sourceDir := detectProfileID(tmpDir)
	finalPath := filepath.Join(gologinDir(), finalID)

	if _, err := os.Stat(finalPath); err == nil {
		if !overwrite {
			return nil, conflict(finalID, fmt.Sprintf("profile %q đã tồn tại, xoá trước khi import lại", finalID))
		}
		_ = os.RemoveAll(finalPath)
	}

	if err := os.MkdirAll(gologinDir(), 0o755); err != nil {
		return nil, err
	}
	if err := moveOrCopy(sourceDir, finalPath); err != nil {
		return nil, err
	}

	ensureProfileName(finalPath, finalID)
	return &ImportResult{ID: finalID}, nil
}

// detectProfileID tìm profileID từ cấu trúc zip đã giải nén.
// Ưu tiên: thư mục con MG-xxx > trường gologin._id trong Preferences > tạo mới.
func detectProfileID(tmpDir string) (id, sourceDir string) {
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "MG-") {
			return e.Name(), filepath.Join(tmpDir, e.Name())
		}
	}

	// Thử đọc từ Preferences
	prefsPath := filepath.Join(tmpDir, "Default", "Preferences")
	if data, err := os.ReadFile(prefsPath); err == nil {
		var prefs struct {
			Gologin struct {
				ID string `json:"_id"`
			} `json:"gologin"`
		}
		if err := json.Unmarshal(data, &prefs); err == nil && strings.HasPrefix(prefs.Gologin.ID, "MG-") {
			return prefs.Gologin.ID, tmpDir
		}
	}

	return "MG-" + util.NewULID(), tmpDir
}

// moveOrCopy thử Rename (nhanh), nếu cross-device thì Copy.
func moveOrCopy(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	return copyDir(src, dst)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func ensureProfileName(profilePath, id string) {
	prefsPath := filepath.Join(profilePath, "Default", "Preferences")
	prefs, err := readPrefsMap(prefsPath)
	if err != nil {
		return
	}
	gl, ok := prefs["gologin"].(map[string]any)
	if !ok {
		prefs["gologin"] = map[string]any{"name": id}
		_ = writePrefsMap(prefsPath, prefs)
		return
	}
	if _, hasName := gl["name"]; !hasName {
		gl["name"] = id
		_ = writePrefsMap(prefsPath, prefs)
	}
}
