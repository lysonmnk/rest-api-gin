package models

import "time"

// Book merepresentasikan entitas buku di database dan response API.
type Book struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Title        string    `json:"title" binding:"required"`
	Author       string    `json:"author" binding:"required"`
	Category     string    `json:"category" gorm:"default:'Umum'"`
	Description  string    `json:"description"`
	ISBN         string    `json:"isbn"`
	PublishYear  int       `json:"publish_year"`
	Stock        int       `json:"stock"`
	IsBookmarked bool      `json:"is_bookmarked" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// StockDeltaInput merepresentasikan perubahan jumlah stok buku.
type StockDeltaInput struct {
	Delta int `json:"delta"`
}

// BookFilter menampung parameter filter dan pengurutan daftar buku.
type BookFilter struct {
	Author         string
	Search         string
	Category       string
	Sort           string
	BookmarkedOnly bool
	LowStockOnly   bool
}

// CategoryStat merepresentasikan jumlah buku per kategori.
type CategoryStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// BookStats menyimpan ringkasan analitik koleksi buku.
type BookStats struct {
	TotalBooks      int            `json:"total_books"`
	TotalStock      int            `json:"total_stock"`
	BookmarkedCount int            `json:"bookmarked_count"`
	LowStockCount   int            `json:"low_stock_count"`
	Categories      []CategoryStat `json:"categories"`
}
