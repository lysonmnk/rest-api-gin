package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
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

func (m *memBookRepo) FindAll(filter models.BookFilter) ([]models.Book, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	author := strings.TrimSpace(filter.Author)
	category := strings.TrimSpace(filter.Category)
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	out := make([]models.Book, 0)
	for _, b := range m.books {
		if author != "" && !strings.EqualFold(b.Author, author) {
			continue
		}
		if category != "" && !strings.EqualFold(b.Category, category) {
			continue
		}
		if filter.BookmarkedOnly && !b.IsBookmarked {
			continue
		}
		if filter.LowStockOnly && b.Stock > 5 {
			continue
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(b.Title), search) &&
			!strings.Contains(strings.ToLower(b.Author), search) &&
			!strings.Contains(strings.ToLower(b.Category), search) &&
			!strings.Contains(strings.ToLower(b.ISBN), search) &&
			!strings.Contains(strings.ToLower(b.Description), search) {
			continue
		}
		out = append(out, b)
	}
	switch strings.TrimSpace(filter.Sort) {
	case "stock_asc":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Stock < out[j].Stock })
	case "stock_desc":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Stock > out[j].Stock })
	case "title":
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
		})
	case "newest":
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].PublishYear != out[j].PublishYear {
				return out[i].PublishYear > out[j].PublishYear
			}
			return out[i].ID > out[j].ID
		})
	default:
		sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
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

// --- Phase 2: Session Cookie, Edge Cases, & Template Polish ---

func doRequestWithCookies(r *gin.Engine, method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSessionCookieAuthAndLogout(t *testing.T) {
	resetData()
	r := setupRouterWithRepos(testUsers, testBooks)

	wLogin := doRequestWithCookies(r, "POST", "/api/login", `{"username":"lyson","password":"password123"}`, nil)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", wLogin.Code)
	}
	cookies := wLogin.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("login harus menyetel cookie session/auth")
	}

	wBooks := doRequestWithCookies(r, "GET", "/api/books", "", cookies)
	if wBooks.Code != http.StatusOK {
		t.Fatalf("GET /api/books via cookie status = %d, want 200", wBooks.Code)
	}

	wLogout := doRequestWithCookies(r, "POST", "/api/logout", "", cookies)
	if wLogout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want 200", wLogout.Code)
	}

	// Terapkan cookie hasil logout (termasuk MaxAge=-1 yang menghapus cookie)
	active := make(map[string]*http.Cookie)
	for _, ck := range cookies {
		active[ck.Name] = ck
	}
	for _, ck := range wLogout.Result().Cookies() {
		if ck.MaxAge < 0 || ck.Value == "" {
			delete(active, ck.Name)
		} else {
			active[ck.Name] = ck
		}
	}
	var afterCookies []*http.Cookie
	for _, ck := range active {
		afterCookies = append(afterCookies, ck)
	}

	wAfter := doRequestWithCookies(r, "GET", "/api/books", "", afterCookies)
	if wAfter.Code != http.StatusUnauthorized {
		t.Fatalf("akses setelah logout status = %d, want 401", wAfter.Code)
	}
}

func TestBookValidationAndFilterEdgeCases(t *testing.T) {
	resetData()
	token := getTestToken(t)

	if w := doRequest("GET", "/api/books/abc", "", token); w.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d, want 400", w.Code)
	}
	if w := doRequest("PUT", "/api/books/1", `{"title":"Valid","author":"Ayu","stock":-1}`, token); w.Code != http.StatusBadRequest {
		t.Fatalf("negative stock on update status = %d, want 400", w.Code)
	}
	if w := doRequest("POST", "/api/books", `{"title":"   ","author":"Ayu"}`, token); w.Code != http.StatusBadRequest {
		t.Fatalf("whitespace title status = %d, want 400", w.Code)
	}
	if w := doRequest("POST", "/api/books", `{rusak`, token); w.Code != http.StatusBadRequest {
		t.Fatalf("malformed json status = %d, want 400", w.Code)
	}
	wAuthor := doRequest("GET", "/api/books?author=ayu", "", token)
	if wAuthor.Code != http.StatusOK || decode(t, wAuthor)["count"].(float64) != 1 {
		t.Fatalf("author filter gagal: code=%d body=%s", wAuthor.Code, wAuthor.Body.String())
	}
	if w := doRequest("GET", "/api/tidak-ada", "", token); w.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want 404", w.Code)
	}
}

func TestTemplatesPolishAndRegisterFlow(t *testing.T) {
	booksRaw, err := os.ReadFile("templates/books.html")
	if err != nil {
		t.Fatalf("gagal membaca templates/books.html: %v", err)
	}
	booksHTML := string(booksRaw)
	if strings.Contains(booksHTML, "includes('localhost:8080')") {
		t.Fatalf("API_BASE masih hardcoded ke localhost:8080")
	}
	if !strings.Contains(booksHTML, "err.message || 'Gagal memperbarui buku'") {
		t.Fatalf("editBookForm belum menampilkan pesan error JSON dari server")
	}
	if !strings.Contains(booksHTML, "searchInput") || !strings.Contains(booksHTML, "Enter") {
		t.Fatalf("searchInput belum mendukung tombol Enter")
	}

	loginRaw, err := os.ReadFile("templates/login.html")
	if err != nil {
		t.Fatalf("gagal membaca templates/login.html: %v", err)
	}
	if !strings.Contains(string(loginRaw), "/api/register") {
		t.Fatalf("templates/login.html belum memiliki alur registrasi (/api/register)")
	}
}

// --- Phase 3 (RED tests: Bookmarks, Stock Delta, Category Filter, Static Assets) ---

func TestToggleBookmarkAndFilter(t *testing.T) {
	resetData()
	token := getTestToken(t)

	w := doRequest("PATCH", "/api/books/1/bookmark", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("bookmark status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["is_bookmarked"] != true {
		t.Fatalf("is_bookmarked = %v, want true", data["is_bookmarked"])
	}

	wOnly := doRequest("GET", "/api/books?bookmarked=true", "", token)
	if wOnly.Code != http.StatusOK || decode(t, wOnly)["count"].(float64) != 1 {
		t.Fatalf("bookmarked filter gagal: code=%d body=%s", wOnly.Code, wOnly.Body.String())
	}

	w404 := doRequest("PATCH", "/api/books/999/bookmark", "", token)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("bookmark unknown id = %d, want 404", w404.Code)
	}
}

func TestAdjustBookStock(t *testing.T) {
	resetData()
	token := getTestToken(t)

	w := doRequest("PATCH", "/api/books/1/stock", `{"delta":-5}`, token)
	if w.Code != http.StatusOK {
		t.Fatalf("stock adjust status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["stock"].(float64) != 10 {
		t.Fatalf("stock = %v, want 10", data["stock"])
	}

	wBad := doRequest("PATCH", "/api/books/1/stock", `{"delta":-50}`, token)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("negative resulting stock = %d, want 400", wBad.Code)
	}
}

func TestStaticLogoAndFaviconServed(t *testing.T) {
	resetData()
	if w := doRequest("GET", "/static/gin-books-white.svg", "", ""); w.Code != http.StatusOK {
		t.Fatalf("static logo status = %d, want 200", w.Code)
	}
	if w := doRequest("GET", "/static/favicon.ico", "", ""); w.Code != http.StatusOK {
		t.Fatalf("static favicon status = %d, want 200", w.Code)
	}
}

// --- Phase 4 (RED tests: Stats, Sort/LowStock Filter, CSV Export, Security Headers) ---

func TestBookStatsEndpoint(t *testing.T) {
	resetData()
	token := getTestToken(t)

	// Tambah buku stok menipis dan terbookmark
	wCreate := doRequest("POST", "/api/books", `{"title":"Algoritma","author":"Citra","category":"Algoritma","stock":3}`, token)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create book gagal: %d %s", wCreate.Code, wCreate.Body.String())
	}
	_ = doRequest("PATCH", "/api/books/1/bookmark", "", token)

	w := doRequest("GET", "/api/books/stats", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/books/stats = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["total_books"].(float64) != 3 {
		t.Fatalf("total_books = %v, want 3", data["total_books"])
	}
	if data["total_stock"].(float64) != 30 {
		t.Fatalf("total_stock = %v, want 30", data["total_stock"])
	}
	if data["bookmarked_count"].(float64) != 1 {
		t.Fatalf("bookmarked_count = %v, want 1", data["bookmarked_count"])
	}
	if data["low_stock_count"].(float64) != 1 {
		t.Fatalf("low_stock_count = %v, want 1", data["low_stock_count"])
	}
}

func TestBookSortAndLowStockFilter(t *testing.T) {
	resetData()
	token := getTestToken(t)
	_ = doRequest("POST", "/api/books", `{"title":"Algoritma Inti","author":"Citra","category":"Algoritma","stock":2}`, token)

	wLow := doRequest("GET", "/api/books?low_stock=true", "", token)
	if wLow.Code != http.StatusOK || decode(t, wLow)["count"].(float64) != 1 {
		t.Fatalf("low_stock filter gagal: code=%d body=%s", wLow.Code, wLow.Body.String())
	}

	wSort := doRequest("GET", "/api/books?sort=stock_asc", "", token)
	if wSort.Code != http.StatusOK {
		t.Fatalf("sort stock_asc gagal: code=%d", wSort.Code)
	}
	items := decode(t, wSort)["data"].([]any)
	first := items[0].(map[string]any)
	if first["title"] != "Algoritma Inti" {
		t.Fatalf("first book stock_asc = %v, want Algoritma Inti", first["title"])
	}
}

func TestExportBooksCSV(t *testing.T) {
	resetData()
	token := getTestToken(t)
	_ = doRequest("POST", "/api/books", `{"title":"=CMD()","author":"Citra","stock":1}`, token)

	w := doRequest("GET", "/api/books/export", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/books/export = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("Content-Type = %q, want text/csv", w.Header().Get("Content-Type"))
	}
	body := w.Body.String()
	if !strings.Contains(body, "ID,Judul,Penulis,Kategori,ISBN,Tahun,Stok,Bookmark") {
		t.Fatalf("CSV header tidak sesuai: %s", body)
	}
	if strings.Contains(body, ",=CMD(),") || !strings.Contains(body, "'=CMD()") {
		t.Fatalf("CSV tidak menetralkan formula injection: %s", body)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	resetData()
	w := doRequest("GET", "/health", "", "")
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", w.Header().Get("X-Frame-Options"))
	}
}
