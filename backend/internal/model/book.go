package model

import "time"

type Book struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title" gorm:"not null"`
	Author    string    `json:"author"`
	ISBN      string    `json:"isbn" gorm:"uniqueIndex"`
	Publisher string    `json:"publisher"`
	Year      int       `json:"year"`
	Stock     int       `json:"stock" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
