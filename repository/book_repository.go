package repository

import (
	"strings"

	"github.com/lysonmnk/rest-api-gin/models"
	"gorm.io/gorm"
)

// BookRepository mendefinisikan kontrak operasi database untuk entitas Book.
type BookRepository interface {
	FindAll(filter models.BookFilter) ([]models.Book, error)
	FindByID(id uint) (*models.Book, error)
	Create(book *models.Book) error
	Update(book *models.Book) error
	Delete(id uint) (int64, error)
}

type bookRepository struct {
	db *gorm.DB
}

// NewBookRepository membuat instance baru BookRepository.
func NewBookRepository(db *gorm.DB) BookRepository {
	return &bookRepository{db: db}
}

var allowedSortOrders = map[string]string{
	"stock_asc":  "stock asc, id asc",
	"stock_desc": "stock desc, id asc",
	"title":      "LOWER(title) asc, id asc",
	"newest":     "publish_year desc, id desc",
}

// FindAll mengambil daftar buku dengan filter opsional penulis, kategori, bookmark, stok menipis, pencarian, dan pengurutan.
func (r *bookRepository) FindAll(filter models.BookFilter) ([]models.Book, error) {
	var books []models.Book
	query := r.db

	if author := strings.TrimSpace(filter.Author); author != "" {
		query = query.Where("LOWER(author) = LOWER(?)", author)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		query = query.Where("LOWER(category) = LOWER(?)", category)
	}
	if filter.BookmarkedOnly {
		query = query.Where("is_bookmarked = ?", true)
	}
	if filter.LowStockOnly {
		query = query.Where("stock <= ?", 5)
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		pattern := "%" + search + "%"
		query = query.Where(
			"(LOWER(title) LIKE LOWER(?) OR LOWER(author) LIKE LOWER(?) OR LOWER(category) LIKE LOWER(?) OR LOWER(isbn) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?))",
			pattern, pattern, pattern, pattern, pattern,
		)
	}

	orderClause, ok := allowedSortOrders[strings.TrimSpace(filter.Sort)]
	if !ok {
		orderClause = "id asc"
	}

	if err := query.Order(orderClause).Find(&books).Error; err != nil {
		return nil, err
	}

	if books == nil {
		books = make([]models.Book, 0)
	}
	return books, nil
}

// FindByID mencari satu buku berdasarkan ID.
func (r *bookRepository) FindByID(id uint) (*models.Book, error) {
	var book models.Book
	if err := r.db.First(&book, id).Error; err != nil {
		return nil, err
	}
	return &book, nil
}

// Create menyimpan buku baru.
func (r *bookRepository) Create(book *models.Book) error {
	book.ID = 0 // Pastikan auto-increment database
	return r.db.Create(book).Error
}

// Update memperbarui data buku yang ada.
func (r *bookRepository) Update(book *models.Book) error {
	return r.db.Save(book).Error
}

// Delete menghapus buku dan mengembalikan jumlah baris yang terpengaruh.
func (r *bookRepository) Delete(id uint) (int64, error) {
	res := r.db.Delete(&models.Book{}, id)
	return res.RowsAffected, res.Error
}
