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
