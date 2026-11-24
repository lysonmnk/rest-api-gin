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
	gin.SetMode(gin.TestMode) // kurangi log yang berisik saat test
}

// resetData mengembalikan data ke kondisi awal sebelum tiap test.
func resetData() {
	mu.Lock()
	defer mu.Unlock()
	books = []Book{
		{ID: 1, Title: "Belajar Go", Author: "Ayu"},
		{ID: 2, Title: "Web API", Author: "Bima"},
	}
	nextID = 3
}

// doRequest mengirim request tiruan ke router tanpa menjalankan server sungguhan.
func doRequest(method, path, body string) *httptest.ResponseRecorder {
	r := setupRouter()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func TestHealth(t *testing.T) {
	resetData()
	w := doRequest("GET", "/health", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestGetBooks(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/books", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	res := decode(t, w)
	if res["count"].(float64) != 2 {
		t.Fatalf("count = %v, want 2", res["count"])
	}
}

func TestGetBooksFilterByAuthor(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/books?author=ayu", "") // huruf kecil pun cocok
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if decode(t, w)["count"].(float64) != 1 {
		t.Fatalf("filter author seharusnya menghasilkan 1 buku")
	}
}

func TestGetBookByID(t *testing.T) {
	resetData()
	if w := doRequest("GET", "/api/books/1", ""); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w := doRequest("GET", "/api/books/999", ""); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if w := doRequest("GET", "/api/books/abc", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateBook(t *testing.T) {
	resetData()
	w := doRequest("POST", "/api/books", `{"title":"Pemrograman Web","author":"Citra"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["id"].(float64) != 3 {
		t.Fatalf("id = %v, want 3", data["id"])
	}
}

func TestCreateBookValidation(t *testing.T) {
	resetData()
	cases := []string{
		`{"title":"Tanpa Penulis"}`,
		`{"author":"Tanpa Judul"}`,
		`{"title":"   ","author":"X"}`,
		`{rusak`,
	}
	for _, body := range cases {
		w := doRequest("POST", "/api/books", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, w.Code)
		}
		if decode(t, w)["status"] != "error" {
			t.Errorf("body %q: status response seharusnya \"error\"", body)
		}
	}
}

func TestUpdateBook(t *testing.T) {
	resetData()
	w := doRequest("PUT", "/api/books/1", `{"title":"Belajar Go Lanjutan","author":"Ayu"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	data := decode(t, w)["data"].(map[string]any)
	if data["title"] != "Belajar Go Lanjutan" {
		t.Fatalf("title = %v", data["title"])
	}
	if w := doRequest("PUT", "/api/books/999", `{"title":"A","author":"B"}`); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDeleteBook(t *testing.T) {
	resetData()
	if w := doRequest("DELETE", "/api/books/1", ""); w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if w := doRequest("GET", "/api/books/1", ""); w.Code != http.StatusNotFound {
		t.Fatalf("setelah dihapus, status = %d, want 404", w.Code)
	}
	if w := doRequest("DELETE", "/api/books/1", ""); w.Code != http.StatusNotFound {
		t.Fatalf("hapus ulang, status = %d, want 404", w.Code)
	}
}

func TestNoRoute(t *testing.T) {
	resetData()
	w := doRequest("GET", "/api/tidak-ada", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
