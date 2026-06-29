// Package zip cung cấp các thao tác nén/giải nén profile.
package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract giải nén zipData vào destDir.
// Trả về error nếu có entry path traversal (zip slip).
func Extract(zipData []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	for _, f := range r.File {
		if err := extractEntry(f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractEntry(f *zip.File, destDir string) error {
	target := filepath.Join(destDir, filepath.FromSlash(f.Name))

	// Bảo vệ zip-slip attack
	if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("zip slip detected: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// Create nén toàn bộ sourceDir vào buffer ZIP và trả về bytes.
// Các file trong zip có đường dẫn tương đối so với parent của sourceDir.
func Create(sourceDir string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Marker "First Run" — GoLogin convention
	if fw, err := zw.Create("First Run"); err == nil {
		_ = fw
	}

	baseDir := filepath.Dir(sourceDir)

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil // bỏ qua lỗi từng file và thư mục
		}
		rel, _ := filepath.Rel(baseDir, path)
		zipName := strings.ReplaceAll(rel, string(filepath.Separator), "/")

		fw, err := zw.Create(zipName)
		if err != nil {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		_, _ = io.Copy(fw, f)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), zw.Close()
}
