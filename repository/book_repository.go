package repository

import (
	"strings"

	"github.com/lysonmnk/rest-api-gin/models"
	"gorm.io/gorm"
)

// BookRepository mendefinisikan kontrak operasi database untuk entitas Book.
type BookRepository interface {
	FindAll(author string) ([]models.Book, error)
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

// FindAll mengambil daftar buku dengan filter opsional penulis (case-insensitive).
func (r *bookRepository) FindAll(author string) ([]models.Book, error) {
	var books []models.Book
	query := r.db

	author = strings.TrimSpace(author)
	if author != "" {
		query = query.Where("LOWER(author) = LOWER(?)", author)
	}

	if err := query.Order("id asc").Find(&books).Error; err != nil {
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
