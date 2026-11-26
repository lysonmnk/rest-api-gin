package config

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GetEnv membaca nilai environment variable dengan nilai bawaan jika kosong.
func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// GetSecret membaca secret dari env var; di dev/test membuat secret acak jika belum diset.
func GetSecret(key string) []byte {
	if val := os.Getenv(key); val != "" {
		return []byte(val)
	}
	if os.Getenv("GIN_MODE") == "release" {
		log.Fatalf("Environment variable %s wajib diset saat GIN_MODE=release", key)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("Gagal membuat secret acak untuk %s: %v", key, err)
	}
	log.Printf("PERINGATAN: %s belum diset, memakai secret acak sementara untuk sesi dev/test", key)
	return buf
}

// ConnectDB menginisialisasi dan mengembalikan koneksi database PostgreSQL via GORM.
func ConnectDB() *gorm.DB {
	host := GetEnv("DB_HOST", "localhost")
	port := GetEnv("DB_PORT", "5433")
	user := GetEnv("DB_USER", "postgres")
	password := GetEnv("DB_PASSWORD", "postgres")
	dbname := GetEnv("DB_NAME", "gin_books_db")
	sslmode := GetEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbname, port, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database PostgreSQL (%s:%s/%s): %v", host, port, dbname, err)
	}

	return db
}
