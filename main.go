package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ===================================================================
// MODEL DAN DATABASE POSTGRESQL (GORM)
// ===================================================================

// Book adalah model database buku dengan field tambahan (ISBN, PublishYear, Stock, CreatedAt).
type Book struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Title       string    `json:"title" binding:"required"`
	Author      string    `json:"author" binding:"required"`
	ISBN        string    `json:"isbn"`
	PublishYear int       `json:"publish_year"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
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

	// Migrasi otomatis schema tabel books (menambahkan kolom baru jika belum ada)
	if err := db.AutoMigrate(&Book{}); err != nil {
		log.Fatal("Gagal melakukan migrasi tabel: ", err)
	}

	var count int64
	db.Model(&Book{}).Count(&count)
	if count == 0 {
		db.Create(&Book{Title: "Belajar Go", Author: "Ayu", ISBN: "978-602-001", PublishYear: 2023, Stock: 15})
		db.Create(&Book{Title: "Web API", Author: "Bima", ISBN: "978-602-002", PublishYear: 2024, Stock: 10})
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
	input.ISBN = strings.TrimSpace(input.ISBN)

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
	book.ISBN = input.ISBN
	book.PublishYear = input.PublishYear
	book.Stock = input.Stock

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
// ROUTER
// ===================================================================

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(RequestInfo())

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

	r.GET("/", func(c *gin.Context) {
		if _, err := os.Stat("index.html"); err == nil {
			c.File("index.html")
		} else {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Gin Books API"})
		}
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
