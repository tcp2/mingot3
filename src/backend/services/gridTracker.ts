// Global array to track running profiles by their assigned slot index
// A slot contains the profile ID string, or null if it's empty.
export const activeSlots: (string | null)[] = [];

export function allocateSlot(id: string): number {
	// Kiểm tra xem profile này đã có slot chưa (nếu lỡ click start 2 lần)
	const existingIdx = activeSlots.indexOf(id);
	if (existingIdx !== -1) return existingIdx;

	// Tìm slot trống đầu tiên
	let slotIndex = activeSlots.indexOf(null);

	if (slotIndex === -1) {
		// Nếu không có slot trống, thêm vào cuối
		slotIndex = activeSlots.length;
		activeSlots.push(id);
	} else {
		// Tái sử dụng slot trống
		activeSlots[slotIndex] = id;
	}

	return slotIndex;
}

export function freeSlot(id: string) {
	const idx = activeSlots.indexOf(id);
	if (idx !== -1) {
		activeSlots[idx] = null;
	}
}

export function getGridConfig(
	slotIndex: number,
	screenWidth: number,
	screenHeight: number,
) {
	// Kích thước mong muốn: 500x480 (không kéo giãn)
	const w = 500;
	const h = 480;

	// Tính toán số cột và hàng tối đa có thể xếp trên màn hình
	const cols = Math.max(1, Math.floor(screenWidth / w));
	const rows = Math.max(1, Math.floor(screenHeight / h));

	// Tính toán dòng, cột
	const col = slotIndex % cols;
	const row = Math.floor(slotIndex / cols) % rows;

	// Cân chỉnh x, y (nếu số thứ tự vượt quá số slot thì nhích x/y một chút để dễ nhìn)
	const offset = Math.floor(slotIndex / (cols * rows)) * 30;

	// Bù trừ viền trong suốt của Windows (thường là 7px mỗi bên lề trái/phải/dưới)
	const winMarginX = 7;
	const winMarginY = 7;

	// Khoảng cách mong muốn giữa các cửa sổ
	const gap = 2;

	return {
		x: col * (w + gap) + offset - winMarginX,
		y: row * (h + gap) + offset,
		w: w + winMarginX * 2,
		h: h + winMarginY,
	};
}
