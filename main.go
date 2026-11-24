package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ===================================================================
// MODEL DAN DATABASE POSTGRESQL (GORM)
// ===================================================================

type Book struct {
	ID     uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Title  string `json:"title" binding:"required"`
	Author string `json:"author" binding:"required"`
}

var db *gorm.DB

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func initDB() {
	if db != nil {
		return
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5433")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "gin_books_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbname, port, sslmode)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal membuka koneksi database PostgreSQL: ", err)
	}

	if err := db.AutoMigrate(&Book{}); err != nil {
		log.Fatal("Gagal melakukan migrasi tabel: ", err)
	}

	var count int64
	db.Model(&Book{}).Count(&count)
	if count == 0 {
		db.Create(&Book{Title: "Belajar Go", Author: "Ayu"})
		db.Create(&Book{Title: "Web API", Author: "Bima"})
		log.Println("Data awal berhasil di-seed ke tabel books di PostgreSQL")
	}
}

// ===================================================================
// FUNGSI BANTU
// ===================================================================

func respondError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"status":  "error",
		"message": message,
	})
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		respondError(c, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return uint(id), true
}

func bindBook(c *gin.Context) (Book, bool) {
	var input Book
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest,
			"Body JSON tidak valid: field title dan author wajib diisi")
		return Book{}, false
	}

	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	if input.Title == "" || input.Author == "" {
		respondError(c, http.StatusBadRequest,
			"title dan author tidak boleh kosong")
		return Book{}, false
	}
	return input, true
}

func RequestInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Request:", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Println("Status:", c.Writer.Status())
	}
}

// ===================================================================
// HANDLER CRUD
// ===================================================================

func getBooks(c *gin.Context) {
	author := strings.TrimSpace(c.Query("author"))
	var result []Book

	query := db
	if author != "" {
		query = query.Where("LOWER(author) = LOWER(?)", author)
	}

	if err := query.Order("id asc").Find(&result).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Gagal mengambil data buku")
		return
	}

	if result == nil {
		result = make([]Book, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(result),
		"data":   result,
	})
}

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

func createBook(c *gin.Context) {
	input, ok := bindBook(c)
	if !ok {
		return
	}

	input.ID = 0
	if err := db.Create(&input).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Gagal menyimpan buku")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": input})
}

func updateBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	input, ok := bindBook(c)
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

	book.Title = input.Title
	book.Author = input.Author
	if err := db.Save(&book).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Gagal memperbarui buku")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": book})
}

func deleteBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	res := db.Delete(&Book{}, id)
	if res.Error != nil {
		respondError(c, http.StatusInternalServerError, "Gagal menghapus buku")
		return
	}
	if res.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "Buku tidak ditemukan")
		return
	}

	c.Status(http.StatusNoContent)
}

// ===================================================================
// ROUTER & TAMPILAN WEB DESKTOP
// ===================================================================

const htmlContent = `<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Gin Books Manager - PostgreSQL</title>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: 'Inter', sans-serif; background: #f1f5f9; color: #1e293b; padding: 24px; }
    .container { max-width: 900px; margin: 0 auto; }
    header { background: linear-gradient(135deg, #1e40af, #3b82f6); color: white; padding: 24px 32px; border-radius: 16px; margin-bottom: 24px; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); display: flex; justify-content: space-between; align-items: center; }
    header h1 { font-size: 24px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
    header p { opacity: 0.9; font-size: 14px; margin-top: 4px; }
    .badge { background: rgba(255,255,255,0.2); padding: 6px 14px; border-radius: 20px; font-size: 13px; font-weight: 600; backdrop-filter: blur(4px); }
    .card { background: white; border-radius: 16px; padding: 24px; margin-bottom: 24px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); }
    .card-title { font-size: 18px; font-weight: 600; margin-bottom: 16px; display: flex; align-items: center; gap: 8px; color: #0f172a; }
    .form-grid { display: grid; grid-template-columns: 1fr 1fr auto; gap: 12px; }
    input[type="text"] { width: 100%; padding: 10px 16px; border: 1.5px solid #cbd5e1; border-radius: 10px; font-size: 14px; font-family: inherit; transition: border-color 0.2s; outline: none; }
    input[type="text"]:focus { border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.15); }
    button { padding: 10px 20px; border-radius: 10px; font-size: 14px; font-weight: 600; font-family: inherit; cursor: pointer; border: none; transition: all 0.2s; display: inline-flex; align-items: center; justify-content: center; gap: 6px; }
    .btn-primary { background: #2563eb; color: white; }
    .btn-primary:hover { background: #1d4ed8; }
    .btn-edit { background: #f59e0b; color: white; padding: 6px 12px; font-size: 12px; border-radius: 6px; }
    .btn-edit:hover { background: #d97706; }
    .btn-delete { background: #ef4444; color: white; padding: 6px 12px; font-size: 12px; border-radius: 6px; margin-left: 6px; }
    .btn-delete:hover { background: #dc2626; }
    .btn-secondary { background: #e2e8f0; color: #334155; }
    .btn-secondary:hover { background: #cbd5e1; }
    .search-box { display: flex; gap: 10px; margin-bottom: 16px; }
    table { width: 100%; border-collapse: collapse; margin-top: 8px; }
    th { text-align: left; padding: 12px 16px; background: #f8fafc; font-size: 13px; font-weight: 600; color: #64748b; border-bottom: 1.5px solid #e2e8f0; }
    td { padding: 14px 16px; border-bottom: 1px solid #f1f5f9; font-size: 14px; }
    tr:hover td { background: #f8fafc; }
    .toast { position: fixed; bottom: 24px; right: 24px; padding: 12px 20px; border-radius: 10px; color: white; font-size: 14px; font-weight: 500; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); display: none; z-index: 1000; }
    .toast.success { background: #10b981; }
    .toast.error { background: #ef4444; }
    .modal { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); display: none; justify-content: center; align-items: center; z-index: 999; }
    .modal-content { background: white; padding: 28px; border-radius: 16px; width: 420px; max-width: 90%; }
    .modal-title { font-size: 18px; font-weight: 700; margin-bottom: 16px; }
    .modal-buttons { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
    .empty-msg { text-align: center; color: #94a3b8; padding: 32px; font-size: 14px; }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div>
        <h1>📚 Gin Books Management</h1>
        <p>Aplikasi CRUD Desktop terhubung langsung ke PostgreSQL (Database: <b>gin_books_db</b>)</p>
      </div>
      <div class="badge">🐘 PostgreSQL 18 Active</div>
    </header>

    <div class="card">
      <div class="card-title">✨ Tambah Buku Baru</div>
      <form id="addBookForm" class="form-grid">
        <input type="text" id="newTitle" placeholder="Judul Buku..." required>
        <input type="text" id="newAuthor" placeholder="Nama Penulis..." required>
        <button type="submit" class="btn-primary">➕ Tambah</button>
      </form>
    </div>

    <div class="card">
      <div class="card-title">📖 Daftar Koleksi Buku</div>
      <div class="search-box">
        <input type="text" id="searchInput" placeholder="Cari berdasarkan nama penulis (contoh: Ayu, Bima)...">
        <button type="button" class="btn-primary" onclick="searchBooks()">Cari</button>
        <button type="button" class="btn-secondary" onclick="resetSearch()">Semua</button>
      </div>

      <table>
        <thead>
          <tr>
            <th style="width: 70px;">ID</th>
            <th>Judul Buku</th>
            <th>Penulis</th>
            <th style="width: 160px; text-align: center;">Aksi</th>
          </tr>
        </thead>
        <tbody id="booksTableBody"></tbody>
      </table>
      <div id="emptyMessage" class="empty-msg" style="display:none;">Tidak ada buku yang ditemukan.</div>
    </div>
  </div>

  <div id="editModal" class="modal">
    <div class="modal-content">
      <div class="modal-title">✏️ Edit Data Buku</div>
      <form id="editBookForm">
        <input type="hidden" id="editBookId">
        <div style="margin-bottom: 14px;">
          <label style="font-size: 13px; font-weight: 600; color: #475569; display:block; margin-bottom: 6px;">Judul Buku</label>
          <input type="text" id="editTitle" required>
        </div>
        <div style="margin-bottom: 14px;">
          <label style="font-size: 13px; font-weight: 600; color: #475569; display:block; margin-bottom: 6px;">Penulis</label>
          <input type="text" id="editAuthor" required>
        </div>
        <div class="modal-buttons">
          <button type="button" class="btn-secondary" onclick="closeEditModal()">Batal</button>
          <button type="submit" class="btn-primary">Simpan Perubahan</button>
        </div>
      </form>
    </div>
  </div>

  <div id="toast" class="toast"></div>

  <script>
    const API_BASE = window.location.origin.includes('localhost:8080') ? '' : 'http://localhost:8080';

    function showToast(msg, type = 'success') {
      const toast = document.getElementById('toast');
      toast.className = 'toast ' + type;
      toast.innerText = msg;
      toast.style.display = 'block';
      setTimeout(() => toast.style.display = 'none', 3000);
    }

    async function loadBooks(authorFilter = '') {
      try {
        let url = API_BASE + '/api/books';
        if (authorFilter) url += '?author=' + encodeURIComponent(authorFilter);
        const res = await fetch(url);
        const data = await res.json();
        const tbody = document.getElementById('booksTableBody');
        tbody.innerHTML = '';

        if (!data.data || data.data.length === 0) {
          document.getElementById('emptyMessage').style.display = 'block';
          return;
        }
        document.getElementById('emptyMessage').style.display = 'none';

        data.data.forEach(book => {
          const tr = document.createElement('tr');
          tr.innerHTML = `
            <td><b>#${book.id}</b></td>
            <td>${book.title}</td>
            <td>${book.author}</td>
            <td style="text-align: center;">
              <button class="btn-edit" onclick="openEditModal(${book.id}, '${book.title.replace(/'/g, "\\'")}', '${book.author.replace(/'/g, "\\'")}')">✏️ Edit</button>
              <button class="btn-delete" onclick="deleteBook(${book.id})">🗑️ Hapus</button>
            </td>
          `;
          tbody.appendChild(tr);
        });
      } catch (err) {
        showToast('Gagal terhubung ke server API (Pastikan go run . sedang berjalan)', 'error');
      }
    }

    document.getElementById('addBookForm').addEventListener('submit', async (e) => {
      e.preventDefault();
      const title = document.getElementById('newTitle').value.trim();
      const author = document.getElementById('newAuthor').value.trim();
      if (!title || !author) return;

      try {
        const res = await fetch(API_BASE + '/api/books', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title, author })
        });
        if (res.ok) {
          showToast('Buku berhasil ditambahkan ke PostgreSQL!');
          document.getElementById('newTitle').value = '';
          document.getElementById('newAuthor').value = '';
          loadBooks();
        } else {
          const err = await res.json();
          showToast(err.message || 'Gagal menambah buku', 'error');
        }
      } catch (err) {
        showToast('Koneksi gagal', 'error');
      }
    });

    async function deleteBook(id) {
      if (!confirm('Apakah Anda yakin ingin menghapus buku #' + id + '?')) return;
      try {
        const res = await fetch(API_BASE + '/api/books/' + id, { method: 'DELETE' });
        if (res.status === 204) {
          showToast('Buku #' + id + ' berhasil dihapus!');
          loadBooks();
        } else {
          showToast('Gagal menghapus buku', 'error');
        }
      } catch (err) {
        showToast('Koneksi gagal', 'error');
      }
    }

    function openEditModal(id, title, author) {
      document.getElementById('editBookId').value = id;
      document.getElementById('editTitle').value = title;
      document.getElementById('editAuthor').value = author;
      document.getElementById('editModal').style.display = 'flex';
    }

    function closeEditModal() {
      document.getElementById('editModal').style.display = 'none';
    }

    document.getElementById('editBookForm').addEventListener('submit', async (e) => {
      e.preventDefault();
      const id = document.getElementById('editBookId').value;
      const title = document.getElementById('editTitle').value.trim();
      const author = document.getElementById('editAuthor').value.trim();

      try {
        const res = await fetch(API_BASE + '/api/books/' + id, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title, author })
        });
        if (res.ok) {
          showToast('Buku #' + id + ' berhasil diperbarui!');
          closeEditModal();
          loadBooks();
        } else {
          showToast('Gagal memperbarui buku', 'error');
        }
      } catch (err) {
        showToast('Koneksi gagal', 'error');
      }
    });

    function searchBooks() {
      const q = document.getElementById('searchInput').value.trim();
      loadBooks(q);
    }

    function resetSearch() {
      document.getElementById('searchInput').value = '';
      loadBooks();
    }

    loadBooks();
  </script>
</body>
</html>`

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(RequestInfo())

	// Middleware CORS agar UI Desktop bisa melakukan fetch tanpa terhalang browser CORS policy
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Endpoint Web UI Desktop
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, htmlContent)
	})

	api := r.Group("/api")
	{
		api.GET("/books", getBooks)
		api.GET("/books/:id", getBookByID)
		api.POST("/books", createBook)
		api.PUT("/books/:id", updateBook)
		api.DELETE("/books/:id", deleteBook)
	}

	r.NoRoute(func(c *gin.Context) {
		respondError(c, http.StatusNotFound, "Endpoint tidak ditemukan")
	})

	return r
}

func main() {
	initDB()

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
