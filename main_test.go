package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lysonmnk/rest-api-gin/config"
	"github.com/lysonmnk/rest-api-gin/middleware"
	"github.com/lysonmnk/rest-api-gin/migrations"
	"github.com/lysonmnk/rest-api-gin/models"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func init() {
	gin.SetMode(gin.TestMode)
	testDB = config.ConnectDB()
	migrations.RunMigrations(testDB)
}

func resetData() {
	testDB.Exec("TRUNCATE TABLE books RESTART IDENTITY")
	testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY")

	adminHash, _ := middleware.HashPassword("admin123")
	lysonHash, _ := middleware.HashPassword("password123")
	testDB.Create(&models.User{Username: "admin", Password: adminHash, FullName: "Administrator", Role: "Admin"})
	testDB.Create(&models.User{Username: "lyson", Password: lysonHash, FullName: "Lyson Manik", Role: "Mahasiswa"})

	testDB.Create(&models.Book{Title: "Belajar Go", Author: "Ayu", ISBN: "978-602-001-A", PublishYear: 2023, Stock: 15})
	testDB.Create(&models.Book{Title: "Web API dengan Gin", Author: "Bima", ISBN: "978-602-002-B", PublishYear: 2024, Stock: 12})
}

func doRequest(method, path, body string, token string) *httptest.ResponseRecorder {
	r := setupRouter(testDB)
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
	var user models.User
	if err := testDB.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("gagal mencari user admin: %v", err)
	}
	token, err := middleware.GenerateToken(user)
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
