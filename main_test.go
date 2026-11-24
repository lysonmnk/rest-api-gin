package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
	initDB()
}

func resetData() {
	db.Exec("TRUNCATE TABLE books RESTART IDENTITY")
	db.Exec("TRUNCATE TABLE users RESTART IDENTITY")

	adminHash, _ := hashPassword("admin123")
	lysonHash, _ := hashPassword("password123")
	db.Create(&User{Username: "admin", Password: adminHash, FullName: "Administrator", Role: "Admin"})
	db.Create(&User{Username: "lyson", Password: lysonHash, FullName: "Lyson Manik", Role: "User"})

	db.Create(&Book{Title: "Belajar Go", Author: "Ayu", ISBN: "978-602-001-A", PublishYear: 2023, Stock: 15})
	db.Create(&Book{Title: "Web API", Author: "Bima", ISBN: "978-602-002-B", PublishYear: 2024, Stock: 12})
}

func doRequest(method, path, body string, token string) *httptest.ResponseRecorder {
	r := setupRouter()
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
	var user User
	if err := db.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("gagal mencari user admin: %v", err)
	}
	token, err := generateToken(user)
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

// 1. Pengujian Alur Login Sukses (Menerima input, verifikasi bcrypt, generate JWT)
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
		t.Fatalf("token seharusnya tidak kosong")
	}
}

// 2. Pengujian Password Salah (Pesan generik mitigasi risiko)
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

// 3. Pengujian User Tidak Ditemukan (Pesan generik sama persis)
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

// 4. Pengujian Validasi Input Format
func TestLoginEmptyValidation(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/login", `{"username":"   ","password":""}`, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// 5. Pengujian Proteksi Route Tanpa Autentikasi (Ditolak 401)
func TestBooksRouteWithoutAuth(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/books", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 Unauthorized saat tanpa token", w.Code)
	}
}

// 6. Pengujian Akses Route Dengan Autentikasi (Diterima 200)
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

// 7. Pengujian Tambah Buku Dengan Autentikasi
func TestCreateBookWithAuth(t *testing.T) {
	resetData()
	token := getTestToken(t)
	w := doRequest("POST", "/api/books", `{"title":"Buku Baru","author":"Penulis Baru","isbn":"978-123","publish_year":2026,"stock":10}`, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["id"].(float64) != 3 {
		t.Fatalf("id = %v, want 3", data["id"])
	}
}

// 8. Pengujian Logout
func TestLogout(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/logout", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}
