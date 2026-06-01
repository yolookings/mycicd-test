# Laporan Audit Keamanan Aplikasi TaskFlow

## Kategori A — Software Composition Analysis (SCA)

### 1. Tool yang Dipilih dan Alasan

Pada kategori Software Composition Analysis (SCA), digunakan **govulncheck** yang dikembangkan langsung oleh Tim Inti Go (Google).

Alasan utama pemilihan tool ini adalah kemampuannya dalam mengurangi jumlah _false positive_ dibandingkan alat pemindai dependensi lainnya seperti Trivy atau Nancy. Berbeda dengan pemindai yang hanya membaca daftar dependensi pada file `go.mod`, govulncheck melakukan analisis terhadap _call graph_ aplikasi untuk menentukan apakah fungsi yang rentan benar-benar dipanggil oleh kode yang sedang diaudit.

Dengan pendekatan tersebut, apabila sebuah pustaka memiliki kerentanan yang telah terdaftar tetapi fungsi yang rentan tidak pernah digunakan oleh aplikasi TaskFlow, maka govulncheck tidak akan menganggapnya sebagai ancaman yang relevan.

### 2. Temuan pada Kode TaskFlow

Pada proses audit, govulncheck melakukan pemindaian terhadap dependensi database PostgreSQL yang digunakan aplikasi, yaitu `github.com/jackc/pgx`.

Jika aplikasi menggunakan versi terbaru dari `pgx` (v5), hasil pemindaian tidak menunjukkan adanya kerentanan yang berdampak pada aplikasi.

Namun, apabila versi dependensi diturunkan ke versi lama yang diketahui memiliki kerentanan seperti _Denial of Service (DoS)_, govulncheck akan:

- Menampilkan informasi kerentanan yang ditemukan.
- Mengidentifikasi fungsi yang memanggil kode rentan tersebut.
- Mengembalikan status kegagalan (_exit status 1_).
- Menghentikan proses CI/CD sehingga rilis tidak dapat dilanjutkan.

### 3. Perbedaan False Positive dan True Positive

#### True Positive

govulncheck mendeteksi bahwa pustaka `pgx` memiliki kerentanan yang diketahui dan kode TaskFlow secara aktif menggunakan fungsi yang terdampak, seperti:

```go
pgx.Connect()
```

atau fungsi lain yang mengeksekusi jalur kode rentan.

#### False Positive

Tool pemindai konvensional dapat melaporkan kerentanan pada pustaka yang terdaftar di `go.mod`, meskipun pustaka tersebut hanya digunakan untuk keperluan pengujian atau bahkan tidak pernah dipanggil oleh aplikasi produksi.

Dalam kondisi tersebut, kerentanan yang dilaporkan tidak memiliki dampak nyata terhadap aplikasi TaskFlow.

### 4. Rekomendasi Perbaikan

Beberapa rekomendasi yang disarankan adalah:

- Melakukan pembaruan versi dependensi secara berkala menggunakan:

```bash
go get -u github.com/jackc/pgx/v5
```

- Mengaktifkan Dependabot pada GitHub agar pembaruan keamanan dapat dilakukan secara otomatis melalui Pull Request.
- Menjadwalkan pemindaian govulncheck secara rutin pada pipeline CI/CD untuk memastikan seluruh dependensi tetap aman.

---

## Kategori B — Static Application Security Testing (SAST)

### 1. Tool yang Dipilih dan Alasan

Pada kategori Static Application Security Testing (SAST), digunakan **gosec** sebagai alat analisis keamanan utama.

gosec bekerja dengan menganalisis Abstract Syntax Tree (AST) dari kode sumber Go dan memiliki berbagai aturan keamanan yang dirancang khusus untuk ekosistem Go. Dengan pendekatan ini, potensi kerentanan dapat ditemukan sebelum kode dikompilasi dan dirilis ke lingkungan produksi.

### 2. Aturan yang Relevan untuk TaskFlow

Dalam konteks aplikasi TaskFlow yang menggunakan Go dan PostgreSQL, beberapa aturan gosec yang paling relevan adalah sebagai berikut.

#### G201 dan G202 — SQL Injection Risk

Aturan ini mendeteksi penggunaan konkatenasi string secara langsung dalam pembuatan query SQL.

Contoh yang tidak aman:

```go
query := "SELECT * FROM tasks WHERE id = " + userInput
```

Pendekatan yang disarankan adalah menggunakan parameterized query:

```go
query := "SELECT * FROM tasks WHERE id = $1"
```

#### G101 — Hardcoded Credentials

Aturan ini mendeteksi kredensial yang dituliskan langsung di dalam kode sumber, seperti:

```go
password := "taskflow_secret"
```

#### G404 — Insecure Random Number Source

Aturan ini mendeteksi penggunaan `math/rand` pada kebutuhan yang bersifat sensitif secara keamanan, seperti token autentikasi atau kode verifikasi.

Untuk kasus tersebut seharusnya digunakan:

```go
crypto/rand
```

### 3. Temuan pada Kode TaskFlow

Selama proses audit menggunakan gosec, beberapa kondisi berikut berpotensi menghasilkan temuan keamanan:

- Query SQL yang dibentuk menggunakan penggabungan string secara dinamis dapat menghasilkan peringatan terkait risiko SQL Injection.
- Kredensial database yang ditulis langsung pada file sumber seperti `main.go` akan terdeteksi sebagai pelanggaran aturan G101.
- Temuan dengan tingkat keparahan tinggi dapat menyebabkan pipeline GitHub Actions gagal dan menghentikan proses deployment.

### 4. Perbedaan False Positive dan True Positive

#### True Positive

gosec menemukan kredensial nyata yang ditulis langsung di dalam kode aplikasi.

Contoh:

```go
password := "12345"
```

Kondisi tersebut merupakan risiko keamanan yang valid karena informasi sensitif tersimpan secara langsung di repository.

#### False Positive

gosec dapat menandai variabel seperti:

```go
mockPassword := "12345"
```

yang sebenarnya hanya digunakan pada unit test atau data simulasi lokal.

Walaupun pola tersebut menyerupai kredensial asli, dalam konteks tertentu temuan tersebut tidak menimbulkan risiko keamanan yang nyata.

### 5. Rekomendasi Perbaikan

Untuk meningkatkan keamanan aplikasi TaskFlow, direkomendasikan langkah-langkah berikut:

- Menyimpan seluruh kredensial melalui environment variable.

Contoh:

```go
databaseURL := os.Getenv("DATABASE_URL")
```

- Menghindari penyimpanan password, token, atau API key secara langsung di dalam kode sumber.
- Menggunakan parameterized query pada seluruh interaksi database PostgreSQL.
- Menggunakan `crypto/rand` untuk kebutuhan yang memerlukan angka acak yang aman secara kriptografi.
- Menggunakan komentar `#nosec` secara terbatas dan hanya setelah temuan diverifikasi secara manual sebagai false positive oleh tim pengembang atau QA.

---

## Kesimpulan

Berdasarkan hasil audit keamanan yang dilakukan, kombinasi penggunaan **govulncheck** untuk Software Composition Analysis (SCA) dan **gosec** untuk Static Application Security Testing (SAST) memberikan cakupan deteksi yang baik terhadap risiko keamanan pada aplikasi TaskFlow.

govulncheck efektif dalam mengidentifikasi kerentanan yang benar-benar digunakan oleh aplikasi melalui analisis call graph, sedangkan gosec mampu mendeteksi pola penulisan kode yang berpotensi menimbulkan celah keamanan sejak tahap pengembangan.

Penerapan kedua alat tersebut secara otomatis pada pipeline CI/CD direkomendasikan untuk memastikan kualitas dan keamanan aplikasi tetap terjaga sebelum setiap proses deployment.
