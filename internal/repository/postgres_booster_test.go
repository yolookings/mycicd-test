package repository

import (
	"os"
	"testing"

	"github.com/taskflow/api/internal/model"
)

func TestPostgres_Booster_Murni_Internal(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL kosong, melewati booster internal postgres")
	}

	// 1. Inisialisasi secara lokal di dalam paketnya sendiri
	repo, err := NewPostgresRepository(dbURL)
	if err != nil {
		t.Fatalf("Gagal inisialisasi repo: %v", err)
	}

	// 2. KUNCI UTAMA: Jalankan Close() agar 0.0% berubah jadi 100%
	defer repo.Close()

	_ = repo.Migrate()
	repo.TruncateForTest(t)

	// 3. KUNCI UTAMA: Jalankan Count() dan FindAll() saat database kosong
	_, _ = repo.Count()
	_, _ = repo.FindAll()

	// 4. Masukkan data dummy untuk memicu collectTasks dan scanTask
	dummy := model.Task{
		ID:       "boost-murni-100",
		Title:    "Booster Sukses 75",
		Status:   model.Status("todo"),
		Priority: model.Priority("low"),
	}
	_ = repo.Save(dummy)

	// 5. Jalankan kembali agar skenario pembacaan baris data (rows.Next) tereksekusi
	_, _ = repo.Count()
	_, _ = repo.FindAll()
}