package migrations

import (
	"log"

	"github.com/lysonmnk/rest-api-gin/middleware"
	"github.com/lysonmnk/rest-api-gin/models"
	"gorm.io/gorm"
)

func hashSeedPassword(password string) string {
	hashed, err := middleware.HashPassword(password)
	if err != nil {
		log.Fatalf("Gagal melakukan hash password seed: %v", err)
	}
	return hashed
}

// RunMigrations menjalankan migrasi skema tabel dan seeding data awal.
func RunMigrations(db *gorm.DB) {
	// 1. Migrasi otomatis skema tabel Book dan User
	if err := db.AutoMigrate(&models.Book{}, &models.User{}); err != nil {
		log.Fatalf("Gagal migrasi skema database: %v", err)
	}

	// 2. Seeding akun default (admin & lyson) jika belum ada
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		db.Create(&models.User{
			Username: "admin",
			Password: hashSeedPassword("admin123"),
			FullName: "Administrator",
			Role:     "Admin",
		})
		db.Create(&models.User{
			Username: "lyson",
			Password: hashSeedPassword("password123"),
			FullName: "Lyson Manik",
			Role:     "Mahasiswa",
		})
		log.Println("Seeding: Akun default admin dan lyson berhasil dibuat.")
	}

	// 3. Seeding data awal koleksi buku jika belum ada
	var bookCount int64
	db.Model(&models.Book{}).Count(&bookCount)
	if bookCount == 0 {
		sampleBooks := []models.Book{
			{Title: "Belajar Go dari Nol", Author: "Ayu Lestari", Category: "Pemrograman", Description: "Panduan praktis memahami sintaks, konkurensi, dan idiomatic Go.", ISBN: "978-602-001-A", PublishYear: 2023, Stock: 15, IsBookmarked: true},
			{Title: "Web API Modern dengan Gin", Author: "Bima Pratama", Category: "Web & API", Description: "Membangun REST API cepat, middleware aman, dan routing terstruktur.", ISBN: "978-602-002-B", PublishYear: 2024, Stock: 12, IsBookmarked: false},
			{Title: "Pemrograman Berbasis Web (PABWE)", Author: "Lyson Manik", Category: "Web & API", Description: "Arsitektur full-stack modern dari backend API hingga antarmuka responsif.", ISBN: "978-623-001-L", PublishYear: 2025, Stock: 25, IsBookmarked: true},
			{Title: "Algoritma & Struktur Data", Author: "Lyson Manik", Category: "Algoritma", Description: "Analisis kompleksitas, graph traversal, dan struktur data inti.", ISBN: "978-623-002-M", PublishYear: 2024, Stock: 4, IsBookmarked: false},
			{Title: "Basis Data Relasional & PostgreSQL", Author: "Lyson Manik", Category: "Basis Data", Description: "Perancangan skema, indexing, dan optimasi query PostgreSQL.", ISBN: "978-623-003-P", PublishYear: 2025, Stock: 30, IsBookmarked: false},
		}
		for _, b := range sampleBooks {
			db.Create(&b)
		}
		log.Println("Seeding: Data koleksi buku awal berhasil dimasukkan.")
	}
}
