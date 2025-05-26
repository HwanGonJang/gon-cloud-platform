package models

import (
	"time"
)

type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	Password     string    `json:"-" db:"password"`
	Role         string    `json:"role" db:"role"` // admin, user
	IsActive     bool      `json:"is_active" db:"is_active"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
