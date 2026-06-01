# 🍏 TaskFlow API - Rollback & Incident Response Strategy

Dokumen ini menjelaskan strategi mitigasi risiko dan prosedur pemulihan bencana (_disaster recovery_) ketika terjadi kegagalan atau ditemukannya _bug_ logika bisnis pasca-rilis kontainer ke lingkungan Staging/Produksi.

---

## 🛠️ Arsitektur Manajemen Rilis

Arsitektur CI/CD kami menerapkan prinsip **Immutable Infrastructure** (Infrastruktur Imutabel) dan penandaan versi yang ketat:

1. **Tag Unik Berbasis Git SHA (`sha-<7-karakter-commit>`)**

   Setiap commit yang lolos pipa CI/CD akan menghasilkan satu _image_ Docker permanen di GitHub Container Registry (GHCR). _Image_ ini tidak akan pernah diubah isinya untuk memastikan pelacakan versi yang presisi (_traceability_).

2. **Tag Dinamis (`stable`)**

   Label penanda versi aman terakhir. Pointer ini hanya akan bergeser ke versi Git SHA terbaru jika dan hanya jika aplikasi lolos dari seluruh tahapan pengujian otomatis (_Unit Test_, _Integration Test_, dan _Smoke Test_).

---

## 🚨 Skenario Kegagalan: "Bug Jumat Sore" (Logic Error)

Skenario ini mensimulasikan kegagalan laten di mana kode aplikasi **lolos dari kompilasi** (pipeline tetap hijau), namun membawa kerusakan fatal pada logika bisnis saat diakses pengguna.

### Contoh Kasus: _Integer Division Bug_

Pada fungsi kalkulasi statistik (`CalculateCompletionRate`), terjadi kecerobohan perubahan tipe data dari pembagian pecahan menjadi pembagian integer murni:

```go
// Mengembalikan hasil pembagian integer yang dipaksa menjadi float (hasil selalu 0)
return float64(completed / len(tasks) * 100)
```

**Dampak:** Pipeline CI/CD tetap Hijau 🟢 karena kode valid secara sintaksis, namun endpoint `/api/v1/stats` mengembalikan nilai `completion_rate_percent: 0` (sistem rusak).

---

## 🚑 Prosedur Pemicuan Rollback (Incident Response)

Ketika tim mendeteksi anomali data pasca-rilis, langkah mitigasi yang diambil bukan melakukan _hotfix_ coding ulang secara terburu-buru, melainkan melakukan pengembalian instan (_instant rollback_) ke versi stabil sebelumnya yang mutlak dijamin aman.

### Langkah-Langkah Eksekusi Rollback Lokal / Server

#### 1. Identifikasi Versi Sehat Terakhir

Buka GitHub Packages (GHCR) atau riwayat commit Git Anda, lalu cari 7 karakter _Short SHA_ dari commit yang berstatus aman sebelum insiden terjadi.

**Contoh versi aman:**

```text
sha-5bbb0a5
```

#### 2. Jalankan Perintah Otomatisasi Rollback

Eksekusi target rollback bawaan melalui Makefile dengan menyertakan variabel `ROLLBACK_TAG`:

```bash
make rollback ROLLBACK_TAG=sha-5bbb0a5
```

---

## ⚙️ Mekanisme Kerja `make rollback` di Balik Layar

Saat perintah di atas dijalankan, target di dalam Makefile akan mengeksekusi otomatisasi orkestrasi kontainer berikut secara berurutan:

```makefile
rollback:
	@test -n "$(ROLLBACK_TAG)" || (echo "❌ Set ROLLBACK_TAG=sha-xxxxx"; exit 1)

	@echo "→ Rolling back ke $(REGISTRY)/$(IMAGE):$(ROLLBACK_TAG)"

	# 1. Menarik image sehat dari GHCR dengan emulasi platform arsitektur host
	docker pull --platform linux/amd64 $(REGISTRY)/$(IMAGE):$(ROLLBACK_TAG)

	# 2. Menghentikan dan menghapus kontainer ber-bug yang sedang berjalan
	docker stop taskflow-api 2>/dev/null || true
	docker rm taskflow-api 2>/dev/null || true

	# 3. Menyalakan ulang kontainer menggunakan image stabil lama & menghubungkannya ke database
	docker run -d \
	  --name taskflow-api \
	  --platform linux/amd64 \
	  -p 8080:8080 \
	  -e DATABASE_URL="postgres://taskflow:taskflow_secret@host.docker.internal:5432/taskflow?sslmode=disable" \
	  $(REGISTRY)/$(IMAGE):$(ROLLBACK_TAG)

	# 4. Melakukan verifikasi kelayakan otomatis (Auto Health Check)
	@echo "⏳ Menunggu server siap..."
	@sleep 5

	curl -sf http://localhost:8080/health || (echo "❌ Health check gagal!"; exit 1)

	@echo "✅ Rollback berhasil ke $(ROLLBACK_TAG)"
```

---

## 🎯 Evaluasi Pasca-Insiden (Post-Mortem)

Setelah layanan berhasil dipulihkan lewat prosedur rollback, tim developer diwajibkan untuk:

1. Memperbaiki logika bisnis asli di lingkungan lokal.

2. Mengaktifkan kembali assertion unit test yang spesifik (menghapus `t.Skip()`) agar bug sejenis tidak lolos lagi di masa mendatang.

3. Melakukan push ulang melalui alur pipa CI/CD yang normal setelah perbaikan terverifikasi murni.
