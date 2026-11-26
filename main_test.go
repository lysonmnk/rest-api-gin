package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lysonmnk/rest-api-gin/middleware"
	"github.com/lysonmnk/rest-api-gin/models"
	"gorm.io/gorm"
)

type memUserRepo struct {
	mu     sync.Mutex
	users  []models.User
	nextID uint
}

func (m *memUserRepo) FindByUsername(username string) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Username == username {
			copyU := u
			return &copyU, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *memUserRepo) FindByID(id uint) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.ID == id {
			copyU := u
			return &copyU, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *memUserRepo) Create(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Username == user.Username {
			return gorm.ErrDuplicatedKey
		}
	}
	user.ID = m.nextID
	m.nextID++
	m.users = append(m.users, *user)
	return nil
}

type memBookRepo struct {
	mu     sync.Mutex
	books  []models.Book
	nextID uint
}

func (m *memBookRepo) FindAll(author, search string) ([]models.Book, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	author = strings.TrimSpace(author)
	search = strings.ToLower(strings.TrimSpace(search))
	out := make([]models.Book, 0)
	for _, b := range m.books {
		if author != "" && !strings.EqualFold(b.Author, author) {
			continue
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(b.Title), search) &&
			!strings.Contains(strings.ToLower(b.Author), search) {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

func (m *memBookRepo) FindByID(id uint) (*models.Book, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.books {
		if b.ID == id {
			copyB := b
			return &copyB, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *memBookRepo) Create(book *models.Book) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	book.ID = m.nextID
	m.nextID++
	m.books = append(m.books, *book)
	return nil
}

func (m *memBookRepo) Update(book *models.Book) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.books {
		if m.books[i].ID == book.ID {
			m.books[i] = *book
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *memBookRepo) Delete(id uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.books {
		if m.books[i].ID == id {
			m.books = append(m.books[:i], m.books[i+1:]...)
			return 1, nil
		}
	}
	return 0, nil
}

var (
	testUsers *memUserRepo
	testBooks *memBookRepo
)

func init() {
	gin.SetMode(gin.TestMode)
}

func resetData() {
	adminHash, _ := middleware.HashPassword("admin123")
	lysonHash, _ := middleware.HashPassword("password123")
	testUsers = &memUserRepo{
		nextID: 3,
		users: []models.User{
			{ID: 1, Username: "admin", Password: adminHash, FullName: "Administrator", Role: "Admin"},
			{ID: 2, Username: "lyson", Password: lysonHash, FullName: "Lyson Manik", Role: "Mahasiswa"},
		},
	}
	testBooks = &memBookRepo{
		nextID: 3,
		books: []models.Book{
			{ID: 1, Title: "Belajar Go", Author: "Ayu", ISBN: "978-602-001-A", PublishYear: 2023, Stock: 15},
			{ID: 2, Title: "Web API dengan Gin", Author: "Bima", ISBN: "978-602-002-B", PublishYear: 2024, Stock: 12},
		},
	}
}

func doRequest(method, path, body string, token string) *httptest.ResponseRecorder {
	r := setupRouterWithRepos(testUsers, testBooks)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("response bukan JSON valid: %v (body: %s)", err, w.Body.String())
	}
	return res
}

func getTestToken(t *testing.T) string {
	user, err := testUsers.FindByUsername("admin")
	if err != nil {
		t.Fatalf("gagal mencari user admin: %v", err)
	}
	token, err := middleware.GenerateToken(*user)
	if err != nil {
		t.Fatalf("gagal generate token: %v", err)
	}
	return token
}

func TestHealth(t *testing.T) {
	resetData()
	w := doRequest("GET", "/health", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestLoginSuccess(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/login", `{"username":"admin","password":"admin123"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	res := decode(t, w)
	if res["status"] != "success" {
		t.Fatalf("status = %v, want success", res["status"])
	}
	if res["token"] == nil || res["token"] == "" {
		t.Fatalf("token tidak boleh kosong")
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/login", `{"username":"admin","password":"password-salah"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	res := decode(t, w)
	if res["message"] != "Username atau password salah" {
		t.Fatalf("message = %v, want generic error", res["message"])
	}
}

func TestLoginUserNotFound(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/login", `{"username":"tidak-ada","password":"password-apapun"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	res := decode(t, w)
	if res["message"] != "Username atau password salah" {
		t.Fatalf("message = %v, want generic error", res["message"])
	}
}

func TestLoginEmptyValidation(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/login", `{"username":"   ","password":""}`, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestBooksRouteWithoutAuth(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/books", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 Unauthorized", w.Code)
	}
}

func TestGetBooksWithAuth(t *testing.T) {
	resetData()
	token := getTestToken(t)
	w := doRequest("GET", "/api/books", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	res := decode(t, w)
	if res["count"].(float64) != 2 {
		t.Fatalf("count = %v, want 2", res["count"])
	}
}

func TestCreateBookWithAuth(t *testing.T) {
	resetData()
	token := getTestToken(t)
	w := doRequest("POST", "/api/books", `{"title":"Buku Modular","author":"Lyson Manik","isbn":"978-000","publish_year":2026,"stock":10}`, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["id"].(float64) != 3 {
		t.Fatalf("id = %v, want 3", data["id"])
	}
}

func TestLogout(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/logout", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// --- Opsi A (RED tests) ---

func TestJWTSecretNotHardcodedPublicString(t *testing.T) {
	resetData()
	tokenStr := getTestToken(t)
	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return []byte("super-secret-jwt-key-pabwe-gin-books-2026"), nil
	})
	if err == nil {
		t.Fatalf("token masih bisa diverifikasi dengan hardcoded secret publik")
	}
}

func TestBooksTemplateAvoidsUnsafeInnerHTMLAndInlineOnclick(t *testing.T) {
	raw, err := os.ReadFile("templates/books.html")
	if err != nil {
		t.Fatalf("gagal membaca templates/books.html: %v", err)
	}
	html := string(raw)
	if strings.Contains(html, "tr.innerHTML") {
		t.Fatalf("templates/books.html masih menggunakan tr.innerHTML yang rentan XSS")
	}
	if strings.Contains(html, "onclick=\"openEditModal(") {
		t.Fatalf("templates/books.html masih menggunakan inline onclick yang rentan kutip/XSS")
	}
}

// --- Opsi B (Validation & Full CRUD / Me tests) ---

func TestCreateBookNegativeStock(t *testing.T) {
	resetData()
	token := getTestToken(t)
	w := doRequest("POST", "/api/books", `{"title":"Buku Minus","author":"Tester","stock":-5}`, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 untuk stock negatif", w.Code)
	}
}

func TestCreateBookInvalidPublishYear(t *testing.T) {
	resetData()
	token := getTestToken(t)
	w := doRequest("POST", "/api/books", `{"title":"Buku Tahun Aneh","author":"Tester","publish_year":1500,"stock":1}`, token)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 untuk publish_year di luar 1900-2100", w.Code)
	}
}

func TestGetBookByIDSuccessAndNotFound(t *testing.T) {
	resetData()
	token := getTestToken(t)

	w := doRequest("GET", "/api/books/1", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["title"] != "Belajar Go" {
		t.Fatalf("title = %v, want Belajar Go", data["title"])
	}

	w404 := doRequest("GET", "/api/books/999", "", token)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w404.Code)
	}
}

func TestUpdateBookSuccessAndNotFound(t *testing.T) {
	resetData()
	token := getTestToken(t)

	w := doRequest("PUT", "/api/books/1", `{"title":"Belajar Go v2","author":"Ayu","isbn":"978-602-001-A","publish_year":2026,"stock":20}`, token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["title"] != "Belajar Go v2" || data["stock"].(float64) != 20 {
		t.Fatalf("data update tidak sesuai: %+v", data)
	}

	w404 := doRequest("PUT", "/api/books/999", `{"title":"X","author":"Y","stock":1}`, token)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w404.Code)
	}
}

func TestDeleteBookSuccessAndNotFound(t *testing.T) {
	resetData()
	token := getTestToken(t)

	w := doRequest("DELETE", "/api/books/2", "", token)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}

	w404 := doRequest("DELETE", "/api/books/2", "", token)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 setelah terhapus", w404.Code)
	}
}

func TestMeWithAuth(t *testing.T) {
	resetData()
	token := getTestToken(t)

	w := doRequest("GET", "/api/me", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["username"] != "admin" {
		t.Fatalf("username = %v, want admin", data["username"])
	}
}

// --- Opsi C (RED tests: Register & Search) ---

func TestRegisterSuccessAndLogin(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/register", `{"username":"budi","password":"secret123","full_name":"Budi Santoso"}`, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["username"] != "budi" || data["role"] != "User" {
		t.Fatalf("unexpected register data: %+v", data)
	}
	if _, hasPwd := data["password"]; hasPwd {
		t.Fatalf("password tidak boleh dikembalikan di response register")
	}

	wLogin := doRequest("POST", "/api/login", `{"username":"budi","password":"secret123"}`, "")
	if wLogin.Code != http.StatusOK {
		t.Fatalf("user baru gagal login: status = %d", wLogin.Code)
	}
}

func TestRegisterDuplicateAndValidation(t *testing.T) {
	resetData()
	wDup := doRequest("POST", "/api/register", `{"username":"admin","password":"secret123","full_name":"Fake"}`, "")
	if wDup.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", wDup.Code)
	}

	wShort := doRequest("POST", "/api/register", `{"username":"baru","password":"123"}`, "")
	if wShort.Code != http.StatusBadRequest {
		t.Fatalf("short password status = %d, want 400", wShort.Code)
	}
}

func TestGetBooksSearchQuery(t *testing.T) {
	resetData()
	token := getTestToken(t)
	w := doRequest("GET", "/api/books?q=Gin", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	res := decode(t, w)
	if res["count"].(float64) != 1 {
		t.Fatalf("count = %v, want 1 buku yang judul/penulisnya cocok dengan 'Gin'", res["count"])
	}
}
