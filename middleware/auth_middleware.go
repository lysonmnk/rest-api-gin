package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lysonmnk/rest-api-gin/config"
	"github.com/lysonmnk/rest-api-gin/models"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = config.GetSecret("JWT_SECRET")

// JWTClaims menyimpan payload identitas pengguna di dalam token.
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// HashPassword mengenkripsi password plain text menggunakan Bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash memverifikasi apakah plain password cocok dengan hash Bcrypt.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken membuat token JWT bertanda tangan digital.
func GenerateToken(user models.User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		FullName: user.FullName,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken memvalidasi keabsahan token JWT dan mengembalikan isinya.
func ParseToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing tidak valid")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token tidak valid")
}

// AuthRequired adalah middleware proteksi route yang memverifikasi Session atau JWT.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Cek dari Gin Session
		session := sessions.Default(c)
		sessionUserID := session.Get("user_id")
		if sessionUserID != nil {
			c.Set("userID", sessionUserID)
			c.Set("username", session.Get("username"))
			c.Set("fullName", session.Get("full_name"))
			c.Set("role", session.Get("role"))
			c.Next()
			return
		}

		// 2. Cek dari Cookie auth_token
		var tokenStr string
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenStr = cookie
		}

		// 3. Cek dari Header Authorization: Bearer <token>
		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Jika tidak ada session maupun token
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Akses ditolak: silakan login terlebih dahulu",
			})
			c.Abort()
			return
		}

		claims, err := ParseToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Sesi login tidak valid atau telah kedaluwarsa",
			})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("fullName", claims.FullName)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// CORSMiddleware menangani header Cross-Origin Resource Sharing.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// RequestInfo mencatat log request masuk dan status keluar.
func RequestInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Request:", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Println("Status:", c.Writer.Status())
	}
}
