package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/lysonmnk/rest-api-gin/middleware"
	"github.com/lysonmnk/rest-api-gin/models"
	"github.com/lysonmnk/rest-api-gin/repository"
)

type AuthHandler struct {
	userRepo repository.UserRepository
}

// NewAuthHandler membuat instance baru AuthHandler.
func NewAuthHandler(userRepo repository.UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

// Login memproses autentikasi sesuai diagram alur kuliah:
// 1. POST /login menerima input
// 2. Cek format data
// 3. Query parameterized
// 4. Verifikasi Bcrypt
// 5. Buat Sesi & Token
// 6. Pesan gagal generik jika gagal
func (h *AuthHandler) Login(c *gin.Context) {
	var input models.LoginInput

	// 1 & 2: Cek format input
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Username dan password wajib diisi",
		})
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)
	if input.Username == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Username dan password tidak boleh kosong",
		})
		return
	}

	// 3: Query Parameterized mencari user dari repository
	user, err := h.userRepo.FindByUsername(input.Username)
	if err != nil {
		// 6: Pesan Gagal Generik: mitigasi risiko kebocoran informasi akun
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Username atau password salah",
		})
		return
	}

	// 4: Verifikasi Keamanan mencocokkan password dengan hash Bcrypt
	if !middleware.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Username atau password salah",
		})
		return
	}

	// 5: Sesi & Token: Simpan ke Gin Session
	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)
	session.Set("full_name", user.FullName)
	session.Set("role", user.Role)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Gagal menyimpan session login",
		})
		return
	}

	// Buat token JWT dan simpan di HTTP-Only Cookie
	token, err := middleware.GenerateToken(*user)
	if err == nil {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", token, 3600*24, "/", "", false, true)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Login berhasil",
		"token":   token,
		"data": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// Logout menghapus sesi login dan cookie autentikasi.
func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	_ = session.Save()

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logout berhasil",
	})
}

// LogoutAndRedirect melakukan logout lalu mengalihkan ke /login.
func (h *AuthHandler) LogoutAndRedirect(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	_ = session.Save()

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// Me mengembalikan informasi pengguna yang sedang login saat ini.
func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"id":        c.MustGet("userID"),
			"username":  c.MustGet("username"),
			"full_name": c.MustGet("fullName"),
			"role":      c.MustGet("role"),
		},
	})
}

// ShowLoginPage merender halaman form login.
func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	if cookie, err := c.Cookie("auth_token"); err == nil {
		if _, err := middleware.ParseToken(cookie); err == nil {
			c.Redirect(http.StatusFound, "/")
			return
		}
	}
	c.File("templates/login.html")
}
