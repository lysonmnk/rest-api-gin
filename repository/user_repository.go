package repository

import (
	"github.com/lysonmnk/rest-api-gin/models"
	"gorm.io/gorm"
)

// UserRepository mendefinisikan kontrak operasi database untuk entitas User.
type UserRepository interface {
	FindByUsername(username string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	Create(user *models.User) error
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat instance baru UserRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// FindByUsername mengambil user berdasarkan username dengan query parameterized (Aman dari SQL Injection).
func (r *userRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID mengambil user berdasarkan primary key ID.
func (r *userRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create menyimpan user baru ke database.
func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}
