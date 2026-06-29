// Package storage cung cấp các đường dẫn hệ thống dùng chung và quản lý cấu hình config.json.
package storage

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

//go:embed zero_profile.json
var zeroProfileData []byte

// Root là thư mục userData, được set từ main.go khi khởi động production.
var Root string

// SetRoot đặt thư mục gốc cho toàn bộ app (thường là userData của OS).
func SetRoot(root string) { Root = root }

func base() string {
	if Root != "" {
		return Root
	}
	dir, _ := os.Getwd()
	return dir
}

// GologinDir trả về thư mục chứa tất cả profile.
func GologinDir() string { return filepath.Join(base(), "profiles") }

// ReadZeroProfile parse và trả về nội dung của file template zero_profile.json được nhúng trong file nhị phân.
func ReadZeroProfile() (map[string]any, error) {
	var data map[string]any
	if err := json.Unmarshal(zeroProfileData, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// BrowserExe trả về đường dẫn tới chrome.exe của Orbita.
func BrowserExe() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gologin", "browser", "orbita-browser-149", "chrome.exe")
}

// TokenStoreDirs trả về các thư mục có thể chứa LevelDB token của GoLogin.
func TokenStoreDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".gologin", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("APPDATA"), "Gologin", "Local Storage", "leveldb"),
	}
}

// ─── App Config Management ───────────────────────────────────────────────────

// AppConfig chứa cấu hình của ứng dụng.
type AppConfig struct {
	GoLoginToken string `json:"gologinToken"`
}

var (
	configLock sync.RWMutex
	cachedCfg  *AppConfig
)

func configPath() string {
	return filepath.Join(base(), "config.json")
}

// LoadConfig tải cấu hình ứng dụng. Nếu file không tồn tại, trả về cấu hình trống.
func LoadConfig() *AppConfig {
	configLock.RLock()
	if cachedCfg != nil {
		defer configLock.RUnlock()
		return cachedCfg
	}
	configLock.RUnlock()

	configLock.Lock()
	defer configLock.Unlock()

	if cachedCfg != nil {
		return cachedCfg
	}

	cfg := &AppConfig{}
	data, err := os.ReadFile(configPath())
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}
	cachedCfg = cfg
	return cachedCfg
}

// SaveConfig lưu cấu hình ứng dụng xuống disk.
func SaveConfig(cfg *AppConfig) error {
	configLock.Lock()
	defer configLock.Unlock()

	if err := os.MkdirAll(base(), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath(), data, 0o644)
	if err == nil {
		cachedCfg = cfg
	}
	return err
}

// GetGoLoginToken trả về token GoLogin đã chọn.
func GetGoLoginToken() string {
	return LoadConfig().GoLoginToken
}

// SetGoLoginToken thiết lập và lưu token GoLogin mới.
func SetGoLoginToken(token string) error {
	cfg := LoadConfig()
	cfg.GoLoginToken = token
	return SaveConfig(cfg)
}
