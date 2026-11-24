# Gin Books API

REST API manajemen buku yang dibangun dengan **Go (Golang)** dan **Gin Framework**.
Proyek ini adalah Tugas Mandiri mata kuliah **Pengembangan Aplikasi Berbasis Web**.

| | |
|---|---|
| **Nama** | Lyson Paulus Esar Manik |
| **NIM** | 11S24024 |
| **Mata kuliah** | Pengembangan Aplikasi Berbasis Web |
| **Materi** | Membangun REST API menggunakan Gin Framework |

---

## Daftar Isi

1. [Tentang Proyek](#1-tentang-proyek)
2. [Konsep Dasar (Penjelasan Sederhana)](#2-konsep-dasar-penjelasan-sederhana)
3. [Prasyarat dan Instalasi Go](#3-prasyarat-dan-instalasi-go)
4. [Membuat Proyek dari Nol (Langkah demi Langkah)](#4-membuat-proyek-dari-nol-langkah-demi-langkah)
5. [Kode Lengkap `main.go`](#5-kode-lengkap-maingo)
6. [Menjalankan Aplikasi](#6-menjalankan-aplikasi)
7. [Dokumentasi Endpoint](#7-dokumentasi-endpoint)
8. [Pengujian Manual (curl, PowerShell, Postman)](#8-pengujian-manual)
9. [Pengujian Otomatis (`go test`)](#9-pengujian-otomatis-go-test)
10. [Build dan Cross Compilation](#10-build-dan-cross-compilation)
11. [Penjelasan Kode Bagian demi Bagian](#11-penjelasan-kode-bagian-demi-bagian)
12. [Praktik Aman](#12-praktik-aman)
13. [Troubleshooting (Kesalahan Umum)](#13-troubleshooting-kesalahan-umum)
14. [Jawaban Pertanyaan Refleksi](#14-jawaban-pertanyaan-refleksi)
15. [Bonus: Database Relasional dengan GORM + SQLite](#15-bonus-database-relasional-dengan-gorm--sqlite)
16. [Pengembangan Lanjutan: Struktur Aplikasi yang Lebih Baik](#16-pengembangan-lanjutan-struktur-aplikasi-yang-lebih-baik)
17. [Checklist Pengumpulan Tugas](#17-checklist-pengumpulan-tugas)
18. [Referensi](#18-referensi)

---

## 1. Tentang Proyek

### Tujuan

Membangun API sederhana untuk mengelola koleksi buku dengan operasi **CRUD** lengkap
(Create, Read, Update, Delete).

### Fitur

- ✅ CRUD lengkap untuk resource `Book`
- ✅ Validasi input: `title` dan `author` **wajib diisi** (termasuk menolak teks yang hanya berisi spasi)
- ✅ Filter berdasarkan penulis: `GET /api/books?author=Ayu`
- ✅ Format response **konsisten** untuk sukses, `400`, `404`, dan `500`
- ✅ Endpoint `GET /health` untuk cek status server
- ✅ Middleware: `Logger`, `Recovery`, dan middleware buatan sendiri (`RequestInfo`)
- ✅ Data aman dari tabrakan akses bersamaan (memakai `sync.RWMutex`)
- ✅ Pengujian otomatis dengan `go test`
- ✅ Bonus: panduan menyimpan data ke database relasional (GORM + SQLite)

### Model Data

Resource `Book` memiliki tiga field:

| Field | Tipe | Keterangan |
|---|---|---|
| `id` | integer | Dibuat otomatis oleh server |
| `title` | string | Wajib diisi |
| `author` | string | Wajib diisi |

### Penyimpanan Data

Penyimpanan awal memakai **slice in-memory** supaya fokus ke konsep Gin.

> **Data hilang saat server dihentikan.** Untuk penyimpanan permanen, lihat
> [bagian Bonus](#15-bonus-database-relasional-dengan-gorm--sqlite).

### Teknologi

- Go (versi terbaru dari <https://go.dev/dl/>)
- Gin Framework (`github.com/gin-gonic/gin`, materi memakai v1.12.0)
- Postman atau curl untuk pengujian

---

## 2. Konsep Dasar (Penjelasan Sederhana)

### 2.1 Go vs Gin: apa bedanya?

- **Go (Golang)** adalah **bahasa pemrograman** buatan Google. Programnya dikompilasi
  menjadi file executable.
- **Gin** **bukan** bahasa pemrograman. Gin adalah **framework HTTP** yang dipakai di dalam program Go
  untuk membuat web server dan REST API dengan lebih mudah.

> **Analogi:** Go itu seperti **bahasa memasak** (cara kerja dasar di dapur).
> Gin itu seperti **set peralatan dapur lengkap** yang sudah tertata, sehingga kamu tidak perlu
> membuat kompor dan panci sendiri dari nol.

### 2.2 Analogi Restoran untuk Arsitektur Gin

Bayangkan aplikasi ini adalah sebuah restoran:

| Di restoran | Di Gin | Tugasnya |
|---|---|---|
| Pelanggan | **Client** (browser, Postman, aplikasi mobile) | Mengirim pesanan |
| Pesanan | **HTTP Request** (method, URL, header, body) | Isi permintaan |
| Pelayan yang membaca pesanan lalu mengarahkannya ke dapur yang tepat | **Router** | Mencocokkan method + path ke handler |
| Koki | **Handler** | Memeriksa pesanan, memasak (logika), menyiapkan hidangan |
| Satpam / kasir di pintu masuk | **Middleware** | Mencatat, memeriksa, atau menolak sebelum/sesudah koki bekerja |
| Nampan berisi hidangan + struk | **Response** (status code + header + JSON) | Jawaban untuk pelanggan |
| Buku catatan pelayan | **`*gin.Context`** | Berisi info pesanan dan alat untuk menyusun jawaban |

Alur lengkapnya:

```
Client ──► HTTP Request ──► Gin Router ──► (Middleware) ──► Handler ──► Response JSON ──► Client
```

### 2.3 Apa itu REST API?

REST API adalah cara dua program berkomunikasi lewat HTTP. Prinsipnya:
**method** menyatakan *operasi*, **path** menyatakan *resource* (benda yang dikelola).

| Method | Kapan dipakai | Analogi |
|---|---|---|
| `GET` | Membaca data | Melihat daftar menu |
| `POST` | Membuat data baru | Memesan menu baru |
| `PUT` | Mengganti/memperbarui data | Mengganti isi pesanan |
| `DELETE` | Menghapus data | Membatalkan pesanan |

### 2.4 HTTP Status Code yang Dipakai

| Kode | Arti | Dipakai ketika |
|---|---|---|
| `200 OK` | Berhasil | GET berhasil, update berhasil |
| `201 Created` | Resource baru dibuat | POST berhasil |
| `204 No Content` | Berhasil, tanpa body | DELETE berhasil |
| `400 Bad Request` | Input tidak valid | JSON salah, field kosong, ID bukan angka |
| `404 Not Found` | Tidak ditemukan | ID buku tidak ada / endpoint tidak ada |
| `500 Internal Server Error` | Kesalahan internal server | Terjadi panic (ditangkap `Recovery`) |

### 2.5 Path Parameter vs Query Parameter

| Jenis | Contoh | Fungsi |
|---|---|---|
| Path parameter | `/api/books/12` | **Mengidentifikasi** satu resource (buku nomor 12) |
| Query parameter | `/api/books?author=Ayu` | **Memfilter/mengatur** hasil |

---

## 3. Prasyarat dan Instalasi Go

Yang dibutuhkan:

| Alat | Fungsi |
|---|---|
| **Go** | Bahasa dan toolchain |
| **Visual Studio Code** + ekstensi **Go** (Go Team at Google) | Editor |
| **Terminal** | PowerShell (Windows) / Terminal (Linux, macOS) |
| **Git** | Version control |
| **Postman** atau **curl** | Menguji API |
| Browser | Membuka endpoint GET sederhana |

### 3.1 Windows

1. Buka <https://go.dev/dl/> lalu pilih installer Windows (`.msi`) sesuai arsitektur komputer.
2. Jalankan installer dan ikuti wizard (lokasi default biasanya `C:\Program Files\Go`).
3. **Tutup lalu buka kembali** PowerShell atau Command Prompt.
4. Verifikasi (langkah 3.4).

### 3.2 Linux (Ubuntu / Linux Mint)

1. Unduh arsip Linux dari <https://go.dev/dl/> sesuai versi dan arsitektur.
2. Jika mengganti instalasi manual lama, hapus dulu `/usr/local/go`.
3. Ekstrak ke `/usr/local`, lalu tambahkan Go ke `PATH`.

```bash
# Ganti nama arsip sesuai file yang kamu unduh
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.XX.X.linux-amd64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
source ~/.profile

go version
```

### 3.3 macOS

1. Unduh paket `.pkg` dari <https://go.dev/dl/> (pilih **Apple Silicon/ARM64** atau **Intel/x86-64**).
2. Buka file `.pkg` dan ikuti petunjuk instalasi.
3. Buka terminal baru, lalu periksa `go version`.
4. Agar tool hasil `go install` bisa dipanggil, pastikan `$(go env GOPATH)/bin` ada di `PATH`.

### 3.4 Verifikasi Instalasi

Jalankan dari **terminal baru**:

```bash
go version
go env GOROOT
go env GOPATH
go env GOMOD
```

Jika `go version` menampilkan versi Go, instalasi berhasil.

---

## 4. Membuat Proyek dari Nol (Langkah demi Langkah)

### Langkah 1: Buat folder dan inisialisasi Go Module

```bash
mkdir gin-books-api
cd gin-books-api

go mod init example.com/gin-books-api
```

> 💡 Jika ingin memakai GitHub, ganti `example.com/gin-books-api` dengan
> `github.com/<username-kamu>/gin-books-api`.
>
> 🏷️ **Analogi:** `go.mod` itu seperti **kartu identitas proyek** sekaligus **daftar belanja**
> bahan (dependensi) yang dibutuhkan.

### Langkah 2: Pasang Gin

```bash
go get github.com/gin-gonic/gin
```

### Langkah 3: Buat file `main.go`

Salin kode lengkap dari [bagian 5](#5-kode-lengkap-maingo).

### Langkah 4: Rapikan dependensi

```bash
go mod tidy
```

Perintah ini mengunduh dependensi yang kurang dan membuang yang tidak terpakai.

### Struktur Direktori Minimal

```
gin-books-api/
├── go.mod          # identitas module dan dependensi
├── go.sum          # checksum dependensi yang diunduh (dibuat otomatis)
├── main.go         # entry point, router, handler
├── main_test.go    # pengujian otomatis (opsional tapi disarankan)
└── README.md       # dokumen ini
```

| File | Fungsi |
|---|---|
| `main.go` | Entry point aplikasi, router, dan handler |
| `go.mod` | Identitas module dan dependensi |
| `go.sum` | Checksum dependensi yang diunduh |

---

## 5. Kode Lengkap `main.go`

Salin **seluruh** isi di bawah ke `main.go`. Semua import sudah pas (tidak ada yang tersisa/tidak terpakai).

```go
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// ===================================================================
// MODEL DAN PENYIMPANAN DATA (in-memory)
// ===================================================================

// Book adalah resource utama pada API ini.
type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title" binding:"required"`
	Author string `json:"author" binding:"required"`
}

var (
	books = []Book{
		{ID: 1, Title: "Belajar Go", Author: "Ayu"},
		{ID: 2, Title: "Web API", Author: "Bima"},
	}
	nextID = 3

	// mu melindungi books dan nextID dari akses bersamaan (konkurensi).
	// Gin melayani banyak request secara paralel, jadi data bersama harus dikunci.
	mu sync.RWMutex
)

// ===================================================================
// FUNGSI BANTU (helper) agar format response konsisten
// ===================================================================

// respondError mengirim response error dengan format yang selalu sama.
func respondError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"status":  "error",
		"message": message,
	})
}

// parseID membaca :id dari URL dan memastikan berupa angka positif.
// Jika tidak valid, response 400 langsung dikirim dan ok = false.
func parseID(c *gin.Context) (id int, ok bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return id, true
}

// bindBook membaca body JSON ke struct Book, lalu memvalidasi isinya.
// Jika gagal, response 400 langsung dikirim dan ok = false.
func bindBook(c *gin.Context) (input Book, ok bool) {
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest,
			"Body JSON tidak valid: field title dan author wajib diisi")
		return Book{}, false
	}

	// Validasi aturan tambahan: tag `required` tetap meloloskan " " (spasi saja),
	// jadi kita periksa lagi setelah dirapikan.
	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	if input.Title == "" || input.Author == "" {
		respondError(c, http.StatusBadRequest,
			"title dan author tidak boleh kosong")
		return Book{}, false
	}
	return input, true
}

// ===================================================================
// MIDDLEWARE BUATAN SENDIRI
// ===================================================================

// RequestInfo mencatat method + path sebelum handler berjalan,
// dan status code setelah handler selesai.
func RequestInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Request:", c.Request.Method, c.Request.URL.Path)
		c.Next() // lanjutkan ke middleware/handler berikutnya
		log.Println("Status:", c.Writer.Status())
	}
}

// ===================================================================
// HANDLER CRUD
// ===================================================================

// getBooks -> GET /api/books  (opsional: ?author=nama)
func getBooks(c *gin.Context) {
	author := strings.TrimSpace(c.Query("author"))

	mu.RLock()
	defer mu.RUnlock()

	result := make([]Book, 0) // make(...) supaya hasil kosong menjadi [] bukan null
	for _, b := range books {
		if author == "" || strings.EqualFold(b.Author, author) {
			result = append(result, b)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(result),
		"data":   result,
	})
}

// getBookByID -> GET /api/books/:id
func getBookByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, b := range books {
		if b.ID == id {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": b})
			return
		}
	}
	respondError(c, http.StatusNotFound, "Buku tidak ditemukan")
}

// createBook -> POST /api/books
func createBook(c *gin.Context) {
	input, ok := bindBook(c)
	if !ok {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	input.ID = nextID // ID dibuat server, bukan dari client
	nextID++
	books = append(books, input)

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": input})
}

// updateBook -> PUT /api/books/:id
func updateBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	input, ok := bindBook(c)
	if !ok {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i := range books {
		if books[i].ID == id {
			books[i].Title = input.Title
			books[i].Author = input.Author
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": books[i]})
			return
		}
	}
	respondError(c, http.StatusNotFound, "Buku tidak ditemukan")
}

// deleteBook -> DELETE /api/books/:id
func deleteBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i := range books {
		if books[i].ID == id {
			books = append(books[:i], books[i+1:]...)
			c.Status(http.StatusNoContent) // 204: berhasil, tanpa body
			return
		}
	}
	respondError(c, http.StatusNotFound, "Buku tidak ditemukan")
}

// ===================================================================
// ROUTER
// ===================================================================

// setupRouter dipisahkan dari main() agar mudah diuji dengan httptest.
func setupRouter() *gin.Engine {
	// gin.New() = router kosong, middleware dipasang manual sesuai kebutuhan.
	// (gin.Default() sudah memasang Logger + Recovery otomatis.)
	r := gin.New()
	r.Use(gin.Logger())   // mencatat setiap request
	r.Use(gin.Recovery()) // mencegah server mati saat terjadi panic (balas 500)
	r.Use(RequestInfo())  // middleware buatan sendiri

	// Endpoint cek kesehatan server
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Group: semua endpoint di bawah ini berawalan /api
	api := r.Group("/api")
	{
		api.GET("/books", getBooks)
		api.GET("/books/:id", getBookByID)
		api.POST("/books", createBook)
		api.PUT("/books/:id", updateBook)
		api.DELETE("/books/:id", deleteBook)
	}

	// Endpoint yang tidak terdaftar -> 404 dengan format JSON yang konsisten
	r.NoRoute(func(c *gin.Context) {
		respondError(c, http.StatusNotFound, "Endpoint tidak ditemukan")
	})

	return r
}

func main() {
	// Port bisa diganti lewat environment variable PORT (default 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := setupRouter()
	log.Println("Server berjalan di http://localhost:" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Gagal menjalankan server: ", err)
	}
}
```

---

## 6. Menjalankan Aplikasi

```bash
# Rapikan dependensi (cukup sekali, atau setelah menambah import)
go mod tidy

# Jalankan aplikasi (untuk pengembangan)
go run .
```

Jika berhasil, terminal menampilkan log Gin dan server berjalan di `http://localhost:8080`.

Cek cepat lewat browser atau terminal lain:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

Ganti port (jika 8080 sedang terpakai):

```bash
# Linux/macOS
PORT=9090 go run .

# Windows PowerShell
$env:PORT="9090"; go run .
```

Hentikan server dengan **Ctrl + C**.

---

## 7. Dokumentasi Endpoint

Base URL: `http://localhost:8080`

| Method | Path | Fungsi | Sukses | Gagal |
|---|---|---|---|---|
| `GET` | `/health` | Cek status server | `200` | - |
| `GET` | `/api/books` | Daftar semua buku | `200` | - |
| `GET` | `/api/books?author=Ayu` | Filter berdasarkan penulis | `200` | - |
| `GET` | `/api/books/:id` | Detail satu buku | `200` | `400`, `404` |
| `POST` | `/api/books` | Tambah buku | `201` | `400` |
| `PUT` | `/api/books/:id` | Ubah buku | `200` | `400`, `404` |
| `DELETE` | `/api/books/:id` | Hapus buku | `204` | `400`, `404` |

### Format Response

**Sukses (satu data):**

```json
{
  "status": "success",
  "data": { "id": 1, "title": "Belajar Go", "author": "Ayu" }
}
```

**Sukses (daftar):**

```json
{
  "status": "success",
  "count": 2,
  "data": [
    { "id": 1, "title": "Belajar Go", "author": "Ayu" },
    { "id": 2, "title": "Web API", "author": "Bima" }
  ]
}
```

**Error (selalu berformat sama untuk `400` dan `404`):**

```json
{
  "status": "error",
  "message": "Buku tidak ditemukan"
}
```

### Contoh Body Request (POST dan PUT)

```json
{
  "title": "Pemrograman Web",
  "author": "Citra"
}
```

> ℹField `id` pada body akan **diabaikan**; ID selalu dibuat/dikendalikan oleh server.

---

## 8. Pengujian Manual

Jalankan server di satu terminal (`go run .`), lalu uji dari terminal **lain**.
Catat **status code** dan **response body** tiap request sebagai bukti tugas.

### 8.1 Dengan curl (Linux / macOS / Git Bash)

```bash
# 1. Health check
curl -i http://localhost:8080/health

# 2. Semua buku
curl -i http://localhost:8080/api/books

# 3. Satu buku
curl -i http://localhost:8080/api/books/1

# 4. Filter berdasarkan penulis
curl -i "http://localhost:8080/api/books?author=Ayu"

# 5. Tambah buku
curl -i -X POST http://localhost:8080/api/books \
  -H "Content-Type: application/json" \
  -d '{"title":"Pemrograman Web","author":"Citra"}'

# 6. Ubah buku
curl -i -X PUT http://localhost:8080/api/books/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Belajar Go Lanjutan","author":"Ayu"}'

# 7. Hapus buku
curl -i -X DELETE http://localhost:8080/api/books/1

# 8. Cek lagi setelah dihapus (harus 404)
curl -i http://localhost:8080/api/books/1
```

`-i` menampilkan status code dan header di samping body.

### 8.2 Dengan PowerShell (Windows)

> Di PowerShell Windows, `curl` adalah alias `Invoke-WebRequest`, dan tanda kutip JSON sering bermasalah.
> Pakai `Invoke-RestMethod` agar aman.

```powershell
# Semua buku
Invoke-RestMethod -Uri http://localhost:8080/api/books

# Satu buku
Invoke-RestMethod -Uri http://localhost:8080/api/books/1

# Filter penulis
Invoke-RestMethod -Uri "http://localhost:8080/api/books?author=Ayu"

# Tambah buku
Invoke-RestMethod -Uri http://localhost:8080/api/books -Method Post `
  -ContentType "application/json" `
  -Body '{"title":"Pemrograman Web","author":"Citra"}'

# Ubah buku
Invoke-RestMethod -Uri http://localhost:8080/api/books/1 -Method Put `
  -ContentType "application/json" `
  -Body '{"title":"Belajar Go Lanjutan","author":"Ayu"}'

# Hapus buku
Invoke-RestMethod -Uri http://localhost:8080/api/books/1 -Method Delete
```

Jika ingin melihat **status code** error di PowerShell, gunakan:

```powershell
try { Invoke-RestMethod -Uri http://localhost:8080/api/books/999 }
catch { $_.Exception.Response.StatusCode.value__ }   # 404
```

### 8.3 Dengan Postman

1. Pilih method (GET/POST/PUT/DELETE) dan isi URL, misalnya `http://localhost:8080/api/books`.
2. Untuk POST/PUT: buka tab **Body → raw → JSON**, lalu isi JSON.
3. Klik **Send**, lalu **screenshot** bagian status code dan response body.

### 8.4 Skenario Uji Lengkap (Isi Tabel Ini untuk Laporan)

Jalankan berurutan pada server yang baru dinyalakan.

| No | Request | Body | Status yang Diharapkan | Keterangan |
|---|---|---|---|---|
| 1 | `GET /health` | - | `200` | `{"status":"ok"}` |
| 2 | `GET /api/books` | - | `200` | 2 buku awal, `count: 2` |
| 3 | `GET /api/books/1` | - | `200` | Buku "Belajar Go" |
| 4 | `GET /api/books/999` | - | `404` | Buku tidak ditemukan |
| 5 | `GET /api/books/abc` | - | `400` | ID tidak valid |
| 6 | `GET /api/books?author=Ayu` | - | `200` | Hanya buku milik Ayu |
| 7 | `GET /api/books?author=Tidak Ada` | - | `200` | `count: 0`, `data: []` |
| 8 | `POST /api/books` | `{"title":"Pemrograman Web","author":"Citra"}` | `201` | `id` = 3 |
| 9 | `POST /api/books` | `{"title":"Tanpa Penulis"}` | `400` | author kosong |
| 10 | `POST /api/books` | `{"title":"   ","author":"X"}` | `400` | title hanya spasi |
| 11 | `POST /api/books` | `{rusak` | `400` | JSON tidak valid |
| 12 | `PUT /api/books/1` | `{"title":"Belajar Go Lanjutan","author":"Ayu"}` | `200` | Data berubah |
| 13 | `PUT /api/books/999` | `{"title":"A","author":"B"}` | `404` | Tidak ditemukan |
| 14 | `PUT /api/books/1` | `{"title":""}` | `400` | Validasi gagal |
| 15 | `DELETE /api/books/2` | - | `204` | Tanpa body |
| 16 | `GET /api/books/2` | - | `404` | Sudah terhapus |
| 17 | `DELETE /api/books/2` | - | `404` | Sudah tidak ada |
| 18 | `GET /api/tidak-ada` | - | `404` | Endpoint tidak ditemukan |

---

## 9. Pengujian Otomatis (`go test`)

Buat file `main_test.go` di folder yang sama dengan `main.go`:

```go
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode) // kurangi log yang berisik saat test
}

// resetData mengembalikan data ke kondisi awal sebelum tiap test.
func resetData() {
	mu.Lock()
	defer mu.Unlock()
	books = []Book{
		{ID: 1, Title: "Belajar Go", Author: "Ayu"},
		{ID: 2, Title: "Web API", Author: "Bima"},
	}
	nextID = 3
}

// doRequest mengirim request tiruan ke router tanpa menjalankan server sungguhan.
func doRequest(method, path, body string) *httptest.ResponseRecorder {
	r := setupRouter()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("response bukan JSON valid: %v (body: %s)", err, w.Body.String())
	}
	return res
}

func TestHealth(t *testing.T) {
	resetData()
	w := doRequest("GET", "/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestGetBooks(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/books", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	res := decode(t, w)
	if res["count"].(float64) != 2 {
		t.Fatalf("count = %v, want 2", res["count"])
	}
}

func TestGetBooksFilterByAuthor(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/books?author=ayu", "") // huruf kecil pun cocok
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if decode(t, w)["count"].(float64) != 1 {
		t.Fatalf("filter author seharusnya menghasilkan 1 buku")
	}
}

func TestGetBookByID(t *testing.T) {
	resetData()
	if w := doRequest("GET", "/api/books/1", ""); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w := doRequest("GET", "/api/books/999", ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if w := doRequest("GET", "/api/books/abc", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateBook(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/books", `{"title":"Pemrograman Web","author":"Citra"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["id"].(float64) != 3 {
		t.Fatalf("id = %v, want 3", data["id"])
	}
}

func TestCreateBookValidation(t *testing.T) {
	resetData()
	cases := []string{
		`{"title":"Tanpa Penulis"}`,
		`{"author":"Tanpa Judul"}`,
		`{"title":"   ","author":"X"}`,
		`{rusak`,
	}
	for _, body := range cases {
		w := doRequest("POST", "/api/books", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, w.Code)
		}
		if decode(t, w)["status"] != "error" {
			t.Errorf("body %q: status response seharusnya \"error\"", body)
		}
	}
}

func TestUpdateBook(t *testing.T) {
	resetData()
	w := doRequest("PUT", "/api/books/1", `{"title":"Belajar Go Lanjutan","author":"Ayu"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["title"] != "Belajar Go Lanjutan" {
		t.Fatalf("title = %v", data["title"])
	}
	if w := doRequest("PUT", "/api/books/999", `{"title":"A","author":"B"}`); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDeleteBook(t *testing.T) {
	resetData()
	if w := doRequest("DELETE", "/api/books/1", ""); w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if w := doRequest("GET", "/api/books/1", ""); w.Code != http.StatusNotFound {
		t.Fatalf("setelah dihapus, status = %d, want 404", w.Code)
	}
	if w := doRequest("DELETE", "/api/books/1", ""); w.Code != http.StatusNotFound {
		t.Fatalf("hapus ulang, status = %d, want 404", w.Code)
	}
}

func TestNoRoute(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/tidak-ada", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
```

Jalankan:

```bash
go test ./...          # ringkas
go test -v ./...       # detail setiap test (bagus untuk screenshot bukti)
go test -race ./...    # cek tabrakan akses data (butuh CGO/gcc di sebagian sistem)
go vet ./...           # cek kesalahan umum pada kode
```

---

## 10. Build dan Cross Compilation

`go run` cocok untuk pengembangan; `go build` menghasilkan **executable**.

```bash
go mod tidy
go test ./...

# Build untuk mesin saat ini
go build -o gin-books-api .

# Linux/macOS
./gin-books-api

# Windows PowerShell
.\gin-books-api.exe
```

> Di Windows, `go build -o gin-books-api .` otomatis menghasilkan `gin-books-api.exe`.

### Cross compilation (build untuk OS lain)

```bash
# Windows x86-64
GOOS=windows GOARCH=amd64 go build -o gin-books.exe .

# Linux x86-64
GOOS=linux GOARCH=amd64 go build -o gin-books-linux .

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o gin-books-macos .

# Lihat semua target yang didukung
go tool dist list
```

Di **PowerShell**, variabel environment diatur dengan cara berbeda:

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o gin-books-linux .
Remove-Item Env:GOOS; Remove-Item Env:GOARCH   # kembalikan ke normal
```

Mode release (mengurangi log debug Gin):

```bash
GIN_MODE=release ./gin-books-api
```

---

## 11. Penjelasan Kode Bagian demi Bagian

### Komponen inti Gin

| Kode | Fungsi |
|---|---|
| `gin.Default()` | Membuat router dengan middleware **Logger** dan **Recovery** |
| `gin.New()` | Membuat router **kosong**, middleware dipasang sendiri |
| `r.GET(path, handler)` | Mendaftarkan handler untuk HTTP GET pada path tertentu |
| `*gin.Context` | Membawa info request dan alat untuk membentuk response |
| `c.JSON(status, data)` | Mengirim response JSON beserta status HTTP |
| `c.Param("id")` | Mengambil path parameter (`/books/:id`) |
| `c.Query("author")` | Mengambil query parameter (`?author=...`) |
| `c.DefaultQuery("q", "")` | Query parameter dengan nilai bawaan |
| `c.ShouldBindJSON(&x)` | Mengisi struct dari body JSON + menjalankan validasi `binding` |
| `c.Status(204)` | Mengirim status tanpa body |
| `r.Group("/api")` | Mengelompokkan endpoint dengan awalan yang sama |
| `r.Use(mw)` | Memasang middleware |
| `r.NoRoute(h)` | Handler untuk path yang tidak terdaftar |
| `r.Run(":8080")` | Menjalankan HTTP server pada port 8080 |
| `gin.H{...}` | Singkatan `map[string]any` untuk menyusun JSON dengan ringkas |

### Struct tag

```go
Title string `json:"title" binding:"required"`
```

- `json:"title"` → nama field di JSON adalah `title` (huruf kecil).
- `binding:"required"` → field wajib ada dan tidak boleh kosong. Aturan lain yang umum:
  `min`, `max`, `email`, misalnya `binding:"required,min=3,max=100"`.

### Mengapa `return` setelah mengirim error?

Mengirim response (`c.JSON(...)`) **tidak menghentikan fungsi**. Tanpa `return`, kode di bawahnya
tetap berjalan dan bisa menimbulkan response ganda atau perubahan data yang tidak diinginkan.

### Mengapa memakai `sync.RWMutex`?

Gin menangani banyak request **bersamaan**. Jika dua request mengubah slice `books` di waktu yang sama,
data bisa rusak (*race condition*).

> **Analogi:** `books` itu seperti **papan tulis bersama**. `Lock()` = hanya satu orang boleh
> menulis dan yang lain menunggu. `RLock()` = banyak orang boleh **membaca** bersamaan selama tidak ada yang menulis.

### Validasi format ≠ validasi aturan bisnis

| Jenis | Contoh |
|---|---|
| Validasi format | `required`, `min`, `max`, `email` (lewat tag `binding`) |
| Validasi aturan bisnis | Judul harus unik; tahun terbit tidak boleh melebihi tahun berjalan |

Pada proyek ini, pemeriksaan `strings.TrimSpace` adalah contoh aturan tambahan di luar tag `binding`.

### Cara kerja middleware (`c.Next()`)

```
Request ─► Logger ─► Recovery ─► RequestInfo ─► Handler
                                    │  (log "Request")     │
                                    │  c.Next() ──────────►│
                                    │◄──────── selesai ────┘
                                    └─ (log "Status")
Response ◄──────────────────────────
```

Kode **sebelum** `c.Next()` berjalan **sebelum** handler; kode **sesudahnya** berjalan **setelah** handler selesai.

---

## 12. Praktik Aman

- **Validasi semua input** dari client; jangan percaya data request.
- Gunakan **status code yang sesuai** dan format error yang konsisten.
- **Jangan kirim** stack trace, kredensial, atau detail internal ke client
  (itulah sebabnya pesan error di proyek ini sudah diseragamkan, bukan memakai `err.Error()` mentah).
- Gunakan **HTTPS** saat deployment; simpan secret lewat environment variable/konfigurasi, **jangan** ditulis di kode.
- Tambahkan **autentikasi dan otorisasi** untuk endpoint yang aksesnya terbatas.
- Atur **timeout**, **batas ukuran request**, logging, dan **graceful shutdown** untuk produksi.
- **Jangan simpan** token atau rahasia sensitif ke dalam log.
- Slice global **bukan** penyimpanan produksi; gunakan repository/database.

---

## 13. Troubleshooting (Kesalahan Umum)

| Pesan / Gejala | Penyebab | Solusi |
|---|---|---|
| `go: command not found` / `'go' is not recognized` | Go belum terpasang atau `PATH` belum diperbarui | Pasang Go, **tutup lalu buka ulang terminal**, cek `go version` |
| `cannot find module` / `no required module provides package` | `go.mod`, import path, atau jaringan bermasalah | Cek `go.mod` dan import, pastikan internet aktif, jalankan `go mod tidy` |
| `go: go.mod file not found` | Perintah dijalankan di luar folder proyek | `cd gin-books-api` lalu ulangi, atau jalankan `go mod init ...` |
| `address already in use` / `bind: Only one usage of each socket address` | Port 8080 sedang dipakai | Hentikan proses lain, atau ganti port: `PORT=9090 go run .` |
| `404 Not Found` | Method atau path salah | Cek method (GET/POST/...) dan path, mis. `/api/books` bukan `/books` |
| `400 Bad Request` | JSON salah / field kosong / ID bukan angka | Periksa JSON, header `Content-Type: application/json`, nama field (`title`, `author`) |
| `undefined: ...` | Salah nama fungsi/package/variabel, atau lupa import | Periksa ejaan, import, dan deklarasi |
| `"xxx" imported and not used` | Ada import yang tidak terpakai | Hapus import tersebut |
| `declared and not used` | Ada variabel yang tidak terpakai | Hapus atau pakai variabelnya |
| `redeclared in this block` | Fungsi/variabel dideklarasikan dua kali | Pastikan setiap handler hanya ada **satu** kali di package `main` |
| `syntax error: unexpected ...` | Kurung kurawal/kurung tidak lengkap | Periksa pasangan `{ }` dan `( )` |
| Di PowerShell, `curl` aneh / JSON ditolak | `curl` adalah alias `Invoke-WebRequest` | Pakai `Invoke-RestMethod` atau `curl.exe`, atau Postman |
| Data kembali ke awal setelah restart | Memakai penyimpanan in-memory | Normal; gunakan database (lihat bonus) |
| `[GIN-debug] [WARNING] Running in "debug" mode` | Mode debug aktif | Hanya peringatan; untuk produksi set `GIN_MODE=release` |

Jika masih bingung, jalankan `go vet ./...` untuk melihat kesalahan pada kode.

---

## 14. Jawaban Pertanyaan Refleksi

**1. Apa perbedaan bahasa Go dan Gin Framework?**
Go adalah bahasa pemrograman (compiled, strongly typed, buatan Google). Gin adalah framework HTTP yang
ditulis dengan Go dan dipakai dari program Go untuk membangun web server/REST API. Gin tidak menggantikan
Go maupun protokol HTTP; ia berdiri di atas fondasi `net/http` milik Go.
*Analogi: Go = bahasa memasak, Gin = set peralatan dapur lengkap.*

**2. Apa peran router, handler, dan `gin.Context`?**
- **Router** mencocokkan method + path request ke handler yang tepat.
- **Handler** memvalidasi input, menjalankan logika, dan membentuk response.
- **`gin.Context`** membawa semua informasi request (parameter, header, body) dan menyediakan fungsi untuk
  membuat response (`c.JSON`, `c.Status`, dll.).
*Analogi: router = pelayan yang mengarahkan pesanan, handler = koki, context = buku catatan pesanan.*

**3. Kapan menggunakan GET, POST, PUT, dan DELETE?**
`GET` untuk membaca data, `POST` untuk membuat data baru, `PUT` untuk mengganti/memperbarui data,
`DELETE` untuk menghapus data.

**4. Mengapa `ShouldBindJSON` dan validasi input diperlukan?**
`ShouldBindJSON` mengubah body JSON menjadi struct Go sekaligus menjalankan aturan `binding`. Validasi perlu
karena data dari client **tidak boleh dipercaya**: bisa kosong, salah tipe, atau berbahaya. Tanpa validasi,
data rusak bisa masuk ke sistem dan menimbulkan bug atau celah keamanan.

**5. Apa manfaat middleware Logger dan Recovery?**
- **Logger** mencatat setiap request (method, path, status, durasi) sehingga mudah melacak masalah.
- **Recovery** menangkap `panic` dan membalas `500` sehingga server **tidak mati** hanya karena satu request bermasalah.

**6. Mengapa penyimpanan in-memory tidak cukup untuk aplikasi produksi?**
Data hilang ketika server berhenti/restart, tidak bisa dibagi antar beberapa instance server, terbatas oleh
ukuran memori, dan tidak aman terhadap akses bersamaan tanpa sinkronisasi. Aplikasi produksi memerlukan
database yang permanen, bisa di-*backup*, dan mendukung transaksi.

---

## 15. Bonus: Database Relasional dengan GORM + SQLite

Bonus tugas: simpan data ke database relasional. Contoh ini memakai **GORM** dan **SQLite versi pure-Go**
(`glebarez/sqlite`, tidak butuh compiler C sehingga mudah di Windows).

### Langkah 1: Pasang dependensi

```bash
go get gorm.io/gorm
go get github.com/glebarez/sqlite
go mod tidy
```

### Langkah 2: Ubah model dan penyimpanan

Hapus variabel `books`, `nextID`, dan `mu`, lalu ganti dengan:

```go
import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Book struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	Title  string `json:"title" binding:"required"`
	Author string `json:"author" binding:"required"`
}

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("books.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal membuka database: ", err)
	}
	if err := db.AutoMigrate(&Book{}); err != nil {
		log.Fatal("Gagal migrasi: ", err)
	}
}
```

Panggil `initDB()` di awal `main()` sebelum `setupRouter()`.

### Langkah 3: Ubah isi handler

| Operasi | Sebelumnya (slice) | Dengan GORM |
|---|---|---|
| Daftar + filter | `for ... append` | `db.Where("LOWER(author) = LOWER(?)", author).Find(&result)` (tanpa filter: `db.Find(&result)`) |
| Cari satu | `for ... if b.ID == id` | `err := db.First(&book, id).Error` → jika `errors.Is(err, gorm.ErrRecordNotFound)` balas `404` |
| Tambah | `append(books, input)` | `db.Create(&input)` (ID terisi otomatis) |
| Ubah | ubah elemen slice | `db.First(&book, id)` → ubah field → `db.Save(&book)` |
| Hapus | `append(books[:i], ...)` | `result := db.Delete(&Book{}, id)` → jika `result.RowsAffected == 0` balas `404` |

Contoh `getBookByID` versi database:

```go
func getBookByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var book Book
	if err := db.First(&book, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "Buku tidak ditemukan")
		} else {
			respondError(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": book})
}
```

> Tambahkan `"errors"` ke daftar import. Mutex tidak diperlukan lagi karena database mengurus
> konkurensi. Tambahkan `books.db` ke `.gitignore`. Jika memakai pengujian otomatis, sesuaikan
> `resetData()` (misalnya memakai database SQLite in-memory `file::memory:`).

---

## 16. Pengembangan Lanjutan: Struktur Aplikasi yang Lebih Baik

Pada proyek yang lebih besar, pisahkan tanggung jawab agar mudah diuji dan dirawat:

| Lapisan | Tugas |
|---|---|
| `handler` | Menerima HTTP request, memvalidasi input, membentuk response |
| `service` | Menjalankan aturan bisnis dan orkestrasi proses |
| `repository` | Mengakses database atau sumber data |
| `model` | Mendefinisikan struktur data/domain |
| `config` | Membaca konfigurasi dan environment |

Contoh struktur:

```
gin-books-api/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── handler/
│   ├── model/
│   ├── repository/
│   └── service/
├── go.mod
├── go.sum
└── README.md
```

### Perbandingan singkat dengan framework lain

| Framework | Ciri |
|---|---|
| **Gin** (Go) | Ringan dan eksplisit; fokus API/web server |
| **Laravel** (PHP) | Full-stack; banyak fasilitas siap pakai |
| **Express** (Node.js) | Web framework minimalis untuk Node.js |

Kesamaannya: routing, middleware, request/response, dan dukungan membangun API.
Perbedaannya: bahasa, runtime, ekosistem, konvensi, dan tingkat fasilitas bawaan.

---

## 17. Checklist Pengumpulan Tugas

Sesuai instruksi **Tugas Mandiri**:

- [ ] API manajemen buku dengan operasi **CRUD lengkap**
- [ ] Validasi bahwa **`title` dan `author` wajib diisi**
- [ ] Endpoint **`GET /api/books?author=...`** untuk filter berdasarkan penulis
- [ ] **Response error konsisten** untuk `400` dan `404`
- [ ] **Source code** (`main.go`, `go.mod`, `go.sum`, dan `main_test.go` bila ada)
- [ ] **README** berisi cara menjalankan (dokumen ini)
- [ ] **Bukti pengujian endpoint**: screenshot Postman/curl (status code + response body) untuk skenario di
      [bagian 8.4](#84-skenario-uji-lengkap-isi-tabel-ini-untuk-laporan), dan/atau screenshot `go test -v ./...`
- [ ] **(Bonus)** Data tersimpan di database relasional

Latihan terbimbing yang sebaiknya sudah terlewati:

1. Buat proyek `gin-books-api` dan inisialisasi Go module.
2. Buat endpoint `GET /health` yang mengembalikan JSON status ok.
3. Buat endpoint `GET /api/books` yang menampilkan dua data buku.
4. Tambahkan `GET /api/books/:id` dan tangani ID tidak valid/tidak ditemukan.
5. Uji setiap endpoint dengan Postman atau curl dan catat status code.

Tips sebelum mengumpulkan:

```bash
gofmt -l .        # daftar file yang belum rapi (kosong = sudah rapi)
gofmt -w .        # rapikan otomatis
go vet ./...
go test ./...
go build -o gin-books-api .
```

Buat juga `.gitignore` sederhana:

```
gin-books-api
gin-books-api.exe
gin-books*.exe
gin-books-linux
gin-books-macos
books.db
.vscode/
```

---

## 18. Referensi

- Gin GitHub & release: <https://github.com/gin-gonic/gin>
- Gin documentation: <https://gin-gonic.com/en/docs/>
- Gin Quickstart: <https://gin-gonic.com/en/docs/quickstart/>
- Go download & install: <https://go.dev/dl/> dan <https://go.dev/doc/install>
- Go tutorial: <https://go.dev/doc/tutorial/getting-started>
- Go Modules reference: <https://go.dev/ref/mod>
- Go RESTful web service tutorial: <https://go.dev/doc/tutorial/web-service-gin>

---

*Dibuat sebagai Tugas Mandiri mata kuliah Pengembangan Aplikasi Berbasis Web, berdasarkan materi
"Membangun REST API menggunakan Gin Framework" oleh Arie Satia Dharma, S.T., M.Kom.*
