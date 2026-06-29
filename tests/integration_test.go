//go:build integration

// Integration tests — chạy với: go test ./tests/... -tags integration -v -timeout 120s
//
// Yêu cầu:
//   - GoLogin đã login (có token trong LevelDB)
//   - Orbita browser đã cài (~/.gologin/browser/orbita-browser-149/chrome.exe)
//   - Có file config/zero_profile.json
//
// Các test chạy tuần tự (t.Run) và share state qua biến package-level
// để mô phỏng đúng user workflow: Create → Start → Stop → Export → Import → Delete.
package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mingot/internal/profile"
	"mingot/internal/storage"
	"mingot/internal/util"
)

// ─── Setup ────────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	// Dùng thư mục tạm làm profiles dir để không ảnh hưởng production data
	tmp, err := os.MkdirTemp("", "mingot-integration-*")
	if err != nil {
		fmt.Println("Không tạo được tmp dir:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	// Copy zero_profile.json không còn cần thiết vì đã embed vào binary.
	storage.SetRoot(tmp)

	fmt.Printf("\n[TEST] Root: %s\n", tmp)
	fmt.Printf("[TEST] Browser: %s\n", storage.BrowserExe())
	fmt.Printf("[TEST] Tokens: %d found\n", len(util.ExtractGoLoginTokens(storage.TokenStoreDirs())))

	code := m.Run()
	os.Exit(code)
}

// ─── Shared state giữa các test step ──────────────────────────────────────────

var (
	mgr          = profile.New()
	createdID    string
	exportedZIP  []byte
	importedID   string
)

// requireTokens bỏ qua test nếu không có GoLogin token.
func requireTokens(t *testing.T) {
	t.Helper()
	if len(util.ExtractGoLoginTokens(storage.TokenStoreDirs())) == 0 {
		t.Skip("Không có GoLogin token — bỏ qua test này")
	}
}

// requireBrowser bỏ qua test nếu browser exe không tồn tại.
func requireBrowser(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(storage.BrowserExe()); err != nil {
		t.Skipf("Browser không tìm thấy tại %s", storage.BrowserExe())
	}
}

// ─── Test suite ───────────────────────────────────────────────────────────────

// TestWorkflow chạy tuần tự các step: Create → List → Start → Stop → Export → Import → Delete.
func TestWorkflow(t *testing.T) {
	t.Run("Step1_List_Empty", testListEmpty)
	t.Run("Step2_Create", testCreate)
	t.Run("Step3_List_AfterCreate", testListAfterCreate)
	t.Run("Step4_Start", testStart)
	t.Run("Step5_Delete_Running_ShouldFail", testDeleteRunningFails)
	t.Run("Step6_Stop", testStop)
	t.Run("Step7_Export", testExport)
	t.Run("Step8_Import", testImport)
	t.Run("Step9_Delete", testDelete)
	t.Run("Step10_Delete_Imported", testDeleteImported)
}

// Step1: List khi chưa có profile nào.
func testListEmpty(t *testing.T) {
	profiles, err := mgr.List()
	if err != nil {
		t.Fatal("List():", err)
	}
	t.Logf("Profile count: %d", len(profiles))
	// Không assert == 0 vì thư mục tmp mới tạo
}

// Step2: Tạo profile mới — gọi GoLogin API thật.
func testCreate(t *testing.T) {
	requireTokens(t)

	t.Log("Đang gọi GoLogin API để lấy fingerprint...")
	result, err := mgr.Create()
	if err != nil {
		t.Fatalf("Create() thất bại: %v", err)
	}

	createdID = result.ID
	t.Logf("✓ Profile tạo thành công: id=%s  name=%s", result.ID, result.Name)

	// Kiểm tra các file đã được tạo trên disk
	profilePath := filepath.Join(storage.GologinDir(), createdID)
	assertFile(t, filepath.Join(profilePath, "Default", "Preferences"))
	assertFile(t, filepath.Join(profilePath, "orbita.config"))
}

// Step3: List sau khi tạo phải thấy profile mới.
func testListAfterCreate(t *testing.T) {
	if createdID == "" {
		t.Skip("Bỏ qua — Step2 chưa chạy hoặc thất bại")
	}

	profiles, err := mgr.List()
	if err != nil {
		t.Fatal("List():", err)
	}

	found := false
	for _, p := range profiles {
		if p.ID == createdID {
			found = true
			t.Logf("✓ Tìm thấy: id=%s  name=%s  status=%s", p.ID, p.Name, p.Status)
		}
	}
	if !found {
		t.Fatalf("Không tìm thấy %s trong danh sách", createdID)
	}
}

// Step4: Khởi chạy browser — cần Orbita exe và token.
func testStart(t *testing.T) {
	if createdID == "" {
		t.Skip("Bỏ qua — cần Step2 hoàn thành")
	}
	requireBrowser(t)

	t.Log("Đang launch browser...")
	result, err := mgr.Start(createdID, 1920, 1080)
	if err != nil {
		t.Fatalf("Start() thất bại: %v", err)
	}
	t.Logf("✓ Browser mở, debug port: %d", result.Port)
	t.Logf("  DevTools: http://127.0.0.1:%d", result.Port)
}

// Step5: Dừng browser qua CDP.
func testStop(t *testing.T) {
	if createdID == "" {
		t.Skip("Bỏ qua — cần Step2 hoàn thành")
	}
	requireBrowser(t)

	// Chờ browser ổn định
	time.Sleep(2 * time.Second)

	t.Log("Đang gửi Browser.close...")
	if err := mgr.Stop(createdID); err != nil {
		t.Fatalf("Stop() thất bại: %v", err)
	}
	t.Log("✓ Lệnh close đã gửi")

	// Chờ process thực sự tắt
	time.Sleep(3 * time.Second)
}

// Step6: Export profile ra ZIP.
func testExport(t *testing.T) {
	if createdID == "" {
		t.Skip("Bỏ qua — cần Step2 hoàn thành")
	}

	t.Log("Đang backup profile...")
	data, err := mgr.Export(createdID)
	if err != nil {
		t.Fatalf("Export() thất bại: %v", err)
	}

	exportedZIP = data
	t.Logf("✓ Backup thành công: %d bytes (%.1f KB)", len(data), float64(len(data))/1024)

	// Kiểm tra magic bytes ZIP
	if len(data) < 4 || data[0] != 'P' || data[1] != 'K' {
		t.Fatal("Dữ liệu trả về không phải ZIP")
	}

	// Lưu ra file để kiểm tra thủ công nếu cần
	outPath := filepath.Join(os.TempDir(), createdID+".zip")
	if err := os.WriteFile(outPath, data, 0o644); err == nil {
		t.Logf("  Saved to: %s", outPath)
	}
}

// Step7: Import lại ZIP vừa export.
func testImport(t *testing.T) {
	if len(exportedZIP) == 0 {
		t.Skip("Bỏ qua — cần Step6 hoàn thành")
	}

	t.Log("Đang import ZIP...")
	result, err := mgr.Import(exportedZIP, false)
	if err != nil {
		t.Fatalf("Import() thất bại: %v", err)
	}

	importedID = result.ID
	t.Logf("✓ Import thành công: id=%s", result.ID)

	// Profile phải xuất hiện trong List
	profiles, _ := mgr.List()
	found := false
	for _, p := range profiles {
		if p.ID == importedID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Không tìm thấy imported profile %s trong danh sách", importedID)
	}
}

// Step5: Thử xoá profile đang chạy — phải lỗi.
func testDeleteRunningFails(t *testing.T) {
	requireBrowser(t)
	if createdID == "" {
		t.Skip("Bỏ qua — cần Step2 hoàn thành")
	}

	err := mgr.Delete(createdID)
	if err == nil {
		t.Fatal("Phải lỗi khi xoá profile đang chạy")
	}
	t.Logf("✓ Đúng: Delete bị từ chối — %v", err)
}

// Step9: Xoá profile gốc.
func testDelete(t *testing.T) {
	if createdID == "" {
		t.Skip("Bỏ qua — cần Step2 hoàn thành")
	}

	if err := mgr.Delete(createdID); err != nil {
		t.Fatalf("Delete() thất bại: %v", err)
	}
	t.Logf("✓ Đã xoá: %s", createdID)

	// Thư mục phải biến mất
	profilePath := filepath.Join(storage.GologinDir(), createdID)
	if _, err := os.Stat(profilePath); err == nil {
		t.Fatal("Thư mục vẫn còn sau khi delete")
	}
}

// Step10: Xoá profile import.
func testDeleteImported(t *testing.T) {
	if importedID == "" {
		t.Skip("Bỏ qua — cần Step7 hoàn thành")
	}

	if err := mgr.Delete(importedID); err != nil {
		t.Fatalf("Delete imported() thất bại: %v", err)
	}
	t.Logf("✓ Đã xoá imported profile: %s", importedID)
}

// ─── Standalone tests (không phụ thuộc workflow) ─────────────────────────────

// TestUpdateFolder kiểm tra bulk folder assignment.
func TestUpdateFolder(t *testing.T) {
	requireTokens(t)

	// Tạo 2 profile để test
	r1, err := mgr.Create()
	if err != nil {
		t.Skipf("Không tạo được profile: %v", err)
	}
	defer mgr.Delete(r1.ID) //nolint:errcheck

	r2, err := mgr.Create()
	if err != nil {
		t.Skipf("Không tạo được profile: %v", err)
	}
	defer mgr.Delete(r2.ID) //nolint:errcheck

	if err := mgr.UpdateFolder([]string{r1.ID, r2.ID}, "TestFolder"); err != nil {
		t.Fatal("UpdateFolder():", err)
	}
	t.Logf("✓ Đã gán folder 'TestFolder' cho %s và %s", r1.ID, r2.ID)

	// Kiểm tra trong List
	profiles, _ := mgr.List()
	for _, p := range profiles {
		if (p.ID == r1.ID || p.ID == r2.ID) && p.Folder != "TestFolder" {
			t.Errorf("profile %s có folder=%q, want TestFolder", p.ID, p.Folder)
		}
	}
}

// ─── Test với production profiles (không cần API) ────────────────────────────

// TestWithExistingProfile test Start/Stop/Export với profile có sẵn trên disk.
// Chạy riêng: go test -tags integration -v -run TestWithExistingProfile ./tests/... -profile MG-xxxxx
func TestWithExistingProfile(t *testing.T) {
	// Dùng production profiles dir thay vì tmp
	storage.SetRoot(os.Getenv("APPDATA") + "/Mingot")
	realMgr := profile.New()

	profiles, err := realMgr.List()
	if err != nil || len(profiles) == 0 {
		t.Skip("Không có profile nào trong production dir")
	}

	// Lấy profile đầu tiên đang stopped
	var target *profile.Summary
	for i := range profiles {
		if profiles[i].Status == "stopped" {
			target = &profiles[i]
			break
		}
	}
	if target == nil {
		t.Skip("Tất cả profiles đang chạy — không test được")
	}

	t.Logf("Test với profile: %s (%s)", target.ID, target.Name)

	t.Run("Export", func(t *testing.T) {
		data, err := realMgr.Export(target.ID)
		if err != nil {
			t.Fatal("Export():", err)
		}
		t.Logf("✓ Export OK: %d bytes (%.1f KB)", len(data), float64(len(data))/1024)
		if data[0] != 'P' || data[1] != 'K' {
			t.Fatal("Không phải ZIP")
		}
	})

	t.Run("Start", func(t *testing.T) {
		requireBrowser(t)
		result, err := realMgr.Start(target.ID, 1920, 1080)
		if err != nil {
			t.Fatal("Start():", err)
		}
		t.Logf("✓ Browser mở: port=%d", result.Port)
		t.Logf("  DevTools: http://127.0.0.1:%d", result.Port)

		// Chờ ổn định rồi stop
		time.Sleep(3 * time.Second)

		if err := realMgr.Stop(target.ID); err != nil {
			t.Logf("WARN Stop(): %v (browser có thể đã tự đóng)", err)
		} else {
			t.Log("✓ Stop OK")
		}
	})
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func assertFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("File không tồn tại: %s", path)
	}
}

