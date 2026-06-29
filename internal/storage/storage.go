// Package storage cung cấp các đường dẫn hệ thống dùng chung.
package storage

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
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
