// create.go — Tạo profile mới qua GoLogin API.
package profile

import "mingot/internal/orbita"

// Create khởi tạo profile mới, trả về ID và tên.
func (m *Manager) Create() (*CreateResult, error) {
	l, err := orbita.New("")
	if err != nil {
		return nil, err
	}
	if err := l.Init(); err != nil {
		return nil, err
	}
	if err := l.BuildProfile(); err != nil {
		return nil, err
	}
	return &CreateResult{ID: l.ProfileID, Name: l.ProfileName}, nil
}
