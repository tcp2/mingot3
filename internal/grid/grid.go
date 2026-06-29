// Package grid quản lý slot vị trí cửa sổ trình duyệt trên màn hình.
package grid

import "sync"

const (
	winW       = 500
	winH       = 480
	gap        = 2
	winMarginX = 7 // viền trong suốt Windows
	winMarginY = 7
)

// Config mô tả vị trí và kích thước cửa sổ.
type Config struct {
	X, Y, W, H int
}

// Tracker phân bổ slot cho các profile đang chạy.
type Tracker struct {
	mu    sync.Mutex
	slots []string // "" = trống
}

// New tạo Tracker mới.
func New() *Tracker { return &Tracker{} }

// Allocate cấp slot cho profileID, trả về index.
// Nếu profile đã có slot, trả về index cũ (idempotent).
func (t *Tracker) Allocate(id string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, s := range t.slots {
		if s == id {
			return i
		}
	}
	for i, s := range t.slots {
		if s == "" {
			t.slots[i] = id
			return i
		}
	}
	t.slots = append(t.slots, id)
	return len(t.slots) - 1
}

// Free giải phóng slot của profileID.
func (t *Tracker) Free(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, s := range t.slots {
		if s == id {
			t.slots[i] = ""
			return
		}
	}
}

// Config tính toán vị trí cửa sổ theo slot index và kích thước màn hình.
func (t *Tracker) Config(slotIndex, screenW, screenH int) Config {
	cols := max(1, screenW/winW)
	rows := max(1, screenH/winH)

	col := slotIndex % cols
	row := (slotIndex / cols) % rows
	offset := (slotIndex / (cols * rows)) * 30

	return Config{
		X: col*(winW+gap) + offset - winMarginX,
		Y: row*(winH+gap) + offset,
		W: winW + winMarginX*2,
		H: winH + winMarginY,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
