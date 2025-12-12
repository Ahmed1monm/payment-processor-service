package model

import "time"

// User represents an authenticated user in the system.
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"size:255;not null"`
	Email        string    `json:"email" gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"` // Never expose in JSON
	Role         string    `json:"role,omitempty" gorm:"size:50;default:'user'"` // Optional for future admin features
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
