package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/lysonmnk/rest-api-gin/middleware"
	"github.com/lysonmnk/rest-api-gin/models"
	"github.com/lysonmnk/rest-api-gin/repository"
	"gorm.io/gorm"
)

type BookHandler struct {
	bookRepo repository.BookRepository
}

// NewBookHandler membuat instance baru BookHandler.
func NewBookHandler(bookRepo repository.BookRepository) *BookHandler {
	return &BookHandler{bookRepo: bookRepo}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "ID tidak valid",
		})
		return 0, false
	}
	return uint(id), true
}

func bindBook(c *gin.Context) (models.Book, bool) {
	var input models.Book
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Body JSON tidak valid: field title dan author wajib diisi",
		})
		return models.Book{}, false
	}

	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	input.ISBN = strings.TrimSpace(input.ISBN)

	if input.Title == "" || input.Author == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "title dan author tidak boleh kosong",
		})
		return models.Book{}, false
	}
	if input.Stock < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "stock tidak boleh negatif",
		})
		return models.Book{}, false
	}
	if input.PublishYear != 0 && (input.PublishYear < 1900 || input.PublishYear > 2100) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "publish_year harus antara 1900 dan 2100",
		})
		return models.Book{}, false
	}
	return input, true
}

func respondBookLookupError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Buku tidak ditemukan",
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"status":  "error",
		"message": "Terjadi kesalahan pada server",
	})
}

// GetBooks mengembalikan daftar semua buku (dengan filter opsional ?author=... atau ?q=...).
func (h *BookHandler) GetBooks(c *gin.Context) {
	author := c.Query("author")
	search := c.Query("q")
	books, err := h.bookRepo.FindAll(author, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengambil data buku",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(books),
		"data":   books,
	})
}

// GetBookByID mengambil detail satu buku berdasarkan ID.
func (h *BookHandler) GetBookByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	book, err := h.bookRepo.FindByID(id)
	if err != nil {
		respondBookLookupError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   book,
	})
}

// CreateBook menambahkan data buku baru.
func (h *BookHandler) CreateBook(c *gin.Context) {
	input, ok := bindBook(c)
	if !ok {
		return
	}

	if err := h.bookRepo.Create(&input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menyimpan buku",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   input,
	})
}

// UpdateBook memperbarui data buku.
func (h *BookHandler) UpdateBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	input, ok := bindBook(c)
	if !ok {
		return
	}

	book, err := h.bookRepo.FindByID(id)
	if err != nil {
		respondBookLookupError(c, err)
		return
	}

	book.Title = input.Title
	book.Author = input.Author
	book.ISBN = input.ISBN
	book.PublishYear = input.PublishYear
	book.Stock = input.Stock

	if err := h.bookRepo.Update(book); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal memperbarui buku",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   book,
	})
}

// DeleteBook menghapus buku berdasarkan ID.
func (h *BookHandler) DeleteBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	rowsAffected, err := h.bookRepo.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menghapus buku",
		})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Buku tidak ditemukan",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ShowBooksPage merender tampilan books.html hanya jika sesi aktif.
func (h *BookHandler) ShowBooksPage(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") == nil {
		// Periksa juga cookie JWT sebagai fallback
		cookie, err := c.Cookie("auth_token")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		if _, err := middleware.ParseToken(cookie); err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}
	}
	c.File("templates/books.html")
}
