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
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ===================================================================
// MODEL DATABASE POSTGRESQL (GORM)
// ===================================================================

// Book adalah model database koleksi buku.
type Book struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Title       string    `json:"title" binding:"required"`
	Author      string    `json:"author" binding:"required"`
	ISBN        string    `json:"isbn"`
	PublishYear int       `json:"publish_year"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// User adalah model database untuk pengguna yang terautentikasi.
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"` // Jangan tampilkan hash di response JSON
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

var db *gorm.DB

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// ===================================================================
// KEAMANAN & AUTENTIKASI (BCRYPT & JWT)
// ===================================================================

var jwtSecret = []byte(getEnv("JWT_SECRET", "super-secret-jwt-key-pabwe-gin-books-2026"))

type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// hashPassword menghasilkan hash Bcrypt yang aman dari plain password.
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPasswordHash membandingkan plain password dengan hash Bcrypt.
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateToken membuat token JWT bertanda tangan dengan masa aktif 24 jam.
func generateToken(user User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		FullName: user.FullName,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// parseToken memverifikasi dan membaca claims dari token JWT.
func parseToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing tidak valid")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token tidak valid")
}

// ===================================================================
// INISIALISASI DATABASE & AUTO-MIGRASI
// ===================================================================

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

	// Migrasi schema Book dan User secara otomatis
	if err := db.AutoMigrate(&Book{}, &User{}); err != nil {
		log.Fatal("Gagal melakukan migrasi tabel: ", err)
	}

	// Inisialisasi akun user default jika tabel masih kosong
	var userCount int64
	db.Model(&User{}).Count(&userCount)
	if userCount == 0 {
		adminHash, _ := hashPassword("admin123")
		lysonHash, _ := hashPassword("password123")
		db.Create(&User{Username: "admin", Password: adminHash, FullName: "Administrator", Role: "Admin"})
		db.Create(&User{Username: "lyson", Password: lysonHash, FullName: "Lyson Manik", Role: "Mahasiswa"})
		log.Println("Akun default admin & lyson berhasil di-seed dengan Bcrypt!")
	}

	// Inisialisasi data buku jika tabel masih kosong
	var bookCount int64
	db.Model(&Book{}).Count(&bookCount)
	if bookCount == 0 {
		db.Create(&Book{Title: "Belajar Go", Author: "Ayu", ISBN: "978-602-001-A", PublishYear: 2023, Stock: 15})
		db.Create(&Book{Title: "Web API dengan Gin", Author: "Bima", ISBN: "978-602-002-B", PublishYear: 2024, Stock: 12})
		db.Create(&Book{Title: "Pemrograman Berbasis Web (PABWE)", Author: "Lyson Manik", ISBN: "978-623-001-L", PublishYear: 2025, Stock: 25})
		db.Create(&Book{Title: "Algoritma & Struktur Data", Author: "Lyson Manik", ISBN: "978-623-002-M", PublishYear: 2024, Stock: 18})
		db.Create(&Book{Title: "Basis Data Relasional & PostgreSQL", Author: "Lyson Manik", ISBN: "978-623-003-P", PublishYear: 2025, Stock: 30})
		log.Println("Data awal buku berhasil di-seed ke PostgreSQL")
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

// ===================================================================
// MIDDLEWARE
// ===================================================================

func RequestInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Request:", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Println("Status:", c.Writer.Status())
	}
}

// AuthMiddleware memproteksi route dari akses yang belum terotentikasi.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// 1. Cek dari HTTP-Only Cookie
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenStr = cookie
		}

		// 2. Jika tidak ada di cookie, cek header Authorization: Bearer <token>
		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			respondError(c, http.StatusUnauthorized, "Akses ditolak: silakan login terlebih dahulu")
			c.Abort()
			return
		}

		claims, err := parseToken(tokenStr)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "Sesi login tidak valid atau telah kedaluwarsa")
			c.Abort()
			return
		}

		// Simpan data identitas pengguna di konteks request
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("fullName", claims.FullName)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// ===================================================================
// HANDLER AUTENTIKASI (LOGIN & LOGOUT) SESUAI ALUR KULIAH
// ===================================================================

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func loginHandler(c *gin.Context) {
	var input LoginInput

	// Langkah 1 & 2: Cek Format & Validasi Input
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)
	if input.Username == "" || input.Password == "" {
		respondError(c, http.StatusBadRequest, "Username dan password tidak boleh kosong")
		return
	}

	// Langkah 3: Query Parameterized untuk proteksi SQL Injection
	var user User
	if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
		// Langkah 6: Pesan Gagal Generik (mitigasi kebocoran informasi apakah username/password yang salah)
		respondError(c, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	// Langkah 4: Verifikasi Keamanan Bandingkan Password dengan Hash Bcrypt
	if !checkPasswordHash(input.Password, user.Password) {
		respondError(c, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	// Langkah 5: Buat Sesi Token JWT
	token, err := generateToken(user)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Gagal membuat sesi login")
		return
	}

	// Simpan ke HTTP-Only Cookie agar aman dari serangan XSS
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", token, 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Login berhasil",
		"token":   token,
		"data": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

func logoutHandler(c *gin.Context) {
	// Menghapus cookie auth_token (MaxAge = -1)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logout berhasil",
	})
}

func meHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"id":        c.MustGet("userID"),
			"username":  c.MustGet("username"),
			"full_name": c.MustGet("fullName"),
			"role":      c.MustGet("role"),
		},
	})
}

// ===================================================================
// HANDLER CRUD BUKU (DIPROTEKSI OLEH AUTENTIKASI)
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

	// Middleware CORS
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

	// Halaman Login Publik
	r.GET("/login", func(c *gin.Context) {
		// Jika sudah login dan valid, langsung alihkan ke dashboard
		if cookie, err := c.Cookie("auth_token"); err == nil {
			if _, err := parseToken(cookie); err == nil {
				c.Redirect(http.StatusFound, "/")
				return
			}
		}
		c.File("login.html")
	})

	// Halaman Utama Dashboard (Dijaga oleh Sesi)
	r.GET("/", func(c *gin.Context) {
		cookie, err := c.Cookie("auth_token")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		if _, err := parseToken(cookie); err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		c.File("index.html")
	})

	// Endpoint API Publik untuk Login & Logout
	r.POST("/api/login", loginHandler)
	r.POST("/api/logout", logoutHandler)
	r.GET("/api/logout", func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/login")
	})

	// Endpoint API yang DILINDUNGI (Wajib Login)
	api := r.Group("/api")
	api.Use(AuthMiddleware())
	{
		api.GET("/me", meHandler)
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
