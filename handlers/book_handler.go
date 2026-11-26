package handlers

import (
	"bytes"
	"encoding/csv"
	"errors"
	"net/http"
	"sort"
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
	input.Category = strings.TrimSpace(input.Category)
	input.Description = strings.TrimSpace(input.Description)
	input.ISBN = strings.TrimSpace(input.ISBN)
	if input.Category == "" {
		input.Category = "Umum"
	}

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

func bookFilterFromQuery(c *gin.Context) models.BookFilter {
	return models.BookFilter{
		Author:         c.Query("author"),
		Search:         c.Query("q"),
		Category:       c.Query("category"),
		Sort:           c.Query("sort"),
		BookmarkedOnly: c.Query("bookmarked") == "true",
		LowStockOnly:   c.Query("low_stock") == "true",
	}
}

func sanitizeCSVCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	default:
		return v
	}
}

// GetBooks mengembalikan daftar buku (filter ?author=, ?q=, ?category=, ?bookmarked=true, ?low_stock=true, ?sort=).
func (h *BookHandler) GetBooks(c *gin.Context) {
	books, err := h.bookRepo.FindAll(bookFilterFromQuery(c))
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

// GetStats mengembalikan ringkasan metrik koleksi buku dan distribusi kategori.
func (h *BookHandler) GetStats(c *gin.Context) {
	books, err := h.bookRepo.FindAll(models.BookFilter{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menghitung statistik koleksi",
		})
		return
	}

	catMap := make(map[string]int)
	stats := models.BookStats{TotalBooks: len(books)}
	for _, b := range books {
		stats.TotalStock += b.Stock
		if b.IsBookmarked {
			stats.BookmarkedCount++
		}
		if b.Stock <= 5 {
			stats.LowStockCount++
		}
		cat := strings.TrimSpace(b.Category)
		if cat == "" {
			cat = "Umum"
		}
		catMap[cat]++
	}

	stats.Categories = make([]models.CategoryStat, 0, len(catMap))
	for name, count := range catMap {
		stats.Categories = append(stats.Categories, models.CategoryStat{Name: name, Count: count})
	}
	sort.Slice(stats.Categories, func(i, j int) bool {
		if stats.Categories[i].Count != stats.Categories[j].Count {
			return stats.Categories[i].Count > stats.Categories[j].Count
		}
		return stats.Categories[i].Name < stats.Categories[j].Name
	})

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// ExportBooksCSV mengekspor katalog buku (termasuk filter aktif) ke berkas CSV yang aman dari formula injection.
func (h *BookHandler) ExportBooksCSV(c *gin.Context) {
	books, err := h.bookRepo.FindAll(bookFilterFromQuery(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal mengekspor data buku",
		})
		return
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"ID", "Judul", "Penulis", "Kategori", "ISBN", "Tahun", "Stok", "Bookmark"})
	for _, b := range books {
		cat := strings.TrimSpace(b.Category)
		if cat == "" {
			cat = "Umum"
		}
		bookmarkLabel := "Tidak"
		if b.IsBookmarked {
			bookmarkLabel = "Ya"
		}
		_ = w.Write([]string{
			strconv.FormatUint(uint64(b.ID), 10),
			sanitizeCSVCell(b.Title),
			sanitizeCSVCell(b.Author),
			sanitizeCSVCell(cat),
			sanitizeCSVCell(b.ISBN),
			strconv.Itoa(b.PublishYear),
			strconv.Itoa(b.Stock),
			bookmarkLabel,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menyusun berkas CSV",
		})
		return
	}

	c.Header("Content-Disposition", `attachment; filename="gin-books-katalog.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
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
	book.Category = input.Category
	book.Description = input.Description
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

// ToggleBookmark membalik status bookmark (simpan) sebuah buku.
func (h *BookHandler) ToggleBookmark(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	book, err := h.bookRepo.FindByID(id)
	if err != nil {
		respondBookLookupError(c, err)
		return
	}

	book.IsBookmarked = !book.IsBookmarked
	if err := h.bookRepo.Update(book); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal memperbarui bookmark buku",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   book,
	})
}

// AdjustStock menambah atau mengurangi stok buku berdasarkan delta tanpa mengizinkan stok negatif.
func (h *BookHandler) AdjustStock(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var input models.StockDeltaInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Body JSON tidak valid: field delta harus berupa angka",
		})
		return
	}

	book, err := h.bookRepo.FindByID(id)
	if err != nil {
		respondBookLookupError(c, err)
		return
	}

	if book.Stock+input.Delta < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "stok akhir tidak boleh negatif",
		})
		return
	}

	book.Stock += input.Delta
	if err := h.bookRepo.Update(book); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal memperbarui stok buku",
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
