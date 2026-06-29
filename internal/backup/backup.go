// Package backup xử lý dọn dẹp và nén profile để backup/export.
package backup

import (
	"fmt"
	"os"
	"path/filepath"

	"mingot/internal/zip"
)

// dirsToRemove và filesToRemove là cache/log không cần backup.
var dirsToRemove = []string{
	"Default/Cache", "Default/Code Cache", "Default/GPUCache",
	"Default/Service Worker", "Default/IndexedDB", "Default/DawnCache",
	"Default/fonts_config", "Default/Shared Dictionary/cache",
	"GrShaderCache", "ShaderCache", "Crashpad", "Guest Profile", "System Profile",
}

var filesToRemove = []string{
	"Default/lockfile", "Default/History-journal",
	"Default/Extension Cookies-journal", "Default/Cookies-journal",
	"Default/Web Data-journal", "Default/QuotaManager-journal",
	"Default/Network/Cookies-journal", "Default/Network/Network Persistent State",
	"Default/Network/Reporting and NEL", "Default/Network/Reporting and NEL-journal",
	"Default/Network/TransportSecurity", "Default/Network/TrustTokens",
	"Default/Network/TrustTokens-journal",
	"chrome_debug.log", "font_config_caches", "CrashpadMetrics-active.pma",
	"SingletonCookie", "SingletonLock", "SingletonSocket",
}

// Sanitize xoá cache và file tạm khỏi profilePath trước khi backup.
// Lỗi xoá từng file được bỏ qua (best-effort).
func Sanitize(profilePath string) {
	remove := func(paths []string, isDir bool) {
		for _, rel := range paths {
			full := filepath.Join(profilePath, filepath.FromSlash(rel))
			if _, err := os.Stat(full); err != nil {
				continue
			}
			if isDir {
				_ = os.RemoveAll(full)
			} else {
				_ = os.Remove(full)
			}
		}
	}
	remove(dirsToRemove, true)
	remove(filesToRemove, false)
}

// Pack sanitize profile rồi nén thư mục Default/ thành ZIP bytes.
func Pack(profilePath string) ([]byte, error) {
	defaultDir := filepath.Join(profilePath, "Default")
	if _, err := os.Stat(defaultDir); err != nil {
		return nil, fmt.Errorf("profile thiếu thư mục Default: %w", err)
	}

	Sanitize(profilePath)

	data, err := zip.Create(defaultDir)
	if err != nil {
		return nil, fmt.Errorf("tạo zip thất bại: %w", err)
	}
	return data, nil
}
