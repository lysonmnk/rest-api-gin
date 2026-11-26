package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/lysonmnk/rest-api-gin/config"
	"github.com/lysonmnk/rest-api-gin/handlers"
	"github.com/lysonmnk/rest-api-gin/middleware"
	"github.com/lysonmnk/rest-api-gin/migrations"
	"github.com/lysonmnk/rest-api-gin/repository"
	"gorm.io/gorm"
)

var sessionSecret = config.GetSecret("SESSION_SECRET")

// setupRouter mengonfigurasi routing dan middleware aplikasi dengan database GORM.
func setupRouter(db *gorm.DB) *gin.Engine {
	return setupRouterWithRepos(
		repository.NewUserRepository(db),
		repository.NewBookRepository(db),
	)
}

// setupRouterWithRepos mengonfigurasi routing dan middleware dengan repository yang dapat diuji terisolasi.
func setupRouterWithRepos(userRepo repository.UserRepository, bookRepo repository.BookRepository) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestInfo())
	r.Use(middleware.CORSMiddleware())

	// 1. Session Store (gin-contrib/sessions dengan cookie store)
	store := cookie.NewStore(sessionSecret)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 24, // 1 hari
		HttpOnly: true,
	})
	r.Use(sessions.Sessions("gin_books_session", store))

	// Inisialisasi Handler
	authHandler := handlers.NewAuthHandler(userRepo)
	bookHandler := handlers.NewBookHandler(bookRepo)

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Aset statis logo & favicon
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Halaman UI Web
	r.GET("/login", authHandler.ShowLoginPage)
	r.GET("/", bookHandler.ShowBooksPage)
	r.GET("/books", bookHandler.ShowBooksPage)

	// Endpoint API Autentikasi Publik
	r.POST("/api/register", authHandler.Register)
	r.POST("/api/login", authHandler.Login)
	r.POST("/api/logout", authHandler.Logout)
	r.GET("/api/logout", authHandler.LogoutAndRedirect)

	// Endpoint API CRUD yang DILINDUNGI (Wajib Login & Session / Token Valid)
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	{
		api.GET("/me", authHandler.Me)
		api.GET("/books", bookHandler.GetBooks)
		api.GET("/books/stats", bookHandler.GetStats)
		api.GET("/books/export", bookHandler.ExportBooksCSV)
		api.GET("/books/:id", bookHandler.GetBookByID)
		api.POST("/books", bookHandler.CreateBook)
		api.PUT("/books/:id", bookHandler.UpdateBook)
		api.PATCH("/books/:id/bookmark", bookHandler.ToggleBookmark)
		api.PATCH("/books/:id/stock", bookHandler.AdjustStock)
		api.DELETE("/books/:id", bookHandler.DeleteBook)
	}

	// 404 Route
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Endpoint tidak ditemukan",
		})
	})

	return r
}

func main() {
	// 1. Inisialisasi koneksi database PostgreSQL
	db := config.ConnectDB()

	// 2. Jalankan migrasi skema tabel dan seeding data
	migrations.RunMigrations(db)

	// 3. Konfigurasi router Gin
	r := setupRouter(db)

	port := config.GetEnv("PORT", "8080")
	log.Println("Server Gin berjalan di http://localhost:" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
