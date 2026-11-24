package models

import "time"

// Book merepresentasikan entitas buku di database dan response API.
type Book struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Title       string    `json:"title" binding:"required"`
	Author      string    `json:"author" binding:"required"`
	ISBN        string    `json:"isbn"`
	PublishYear int       `json:"publish_year"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}
