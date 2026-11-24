package migrations

import (
	"log"

	"github.com/lysonmnk/rest-api-gin/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// hashPassword membantu hashing password dengan Bcrypt untuk seeding awal.
func hashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Gagal melakukan hash password seed: %v", err)
	}
	return string(bytes)
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
			Password: hashPassword("admin123"),
			FullName: "Administrator",
			Role:     "Admin",
		})
		db.Create(&models.User{
			Username: "lyson",
			Password: hashPassword("password123"),
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
			{Title: "Belajar Go", Author: "Ayu", ISBN: "978-602-001-A", PublishYear: 2023, Stock: 15},
			{Title: "Web API dengan Gin", Author: "Bima", ISBN: "978-602-002-B", PublishYear: 2024, Stock: 12},
			{Title: "Pemrograman Berbasis Web (PABWE)", Author: "Lyson Manik", ISBN: "978-623-001-L", PublishYear: 2025, Stock: 25},
			{Title: "Algoritma & Struktur Data", Author: "Lyson Manik", ISBN: "978-623-002-M", PublishYear: 2024, Stock: 18},
			{Title: "Basis Data Relasional & PostgreSQL", Author: "Lyson Manik", ISBN: "978-623-003-P", PublishYear: 2025, Stock: 30},
		}
		for _, b := range sampleBooks {
			db.Create(&b)
		}
		log.Println("Seeding: Data koleksi buku awal berhasil dimasukkan.")
	}
}
