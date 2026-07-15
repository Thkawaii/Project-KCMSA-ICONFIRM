package models

import "time"

type User struct {
	ID       uint `gorm:"primaryKey"`

	RoleName string `gorm:"size:50"`

	Username string `gorm:"size:100;unique"`

	Password string `gorm:"size:255"`

	Status string `gorm:"size:20"`

	Name string `gorm:"size:100"`

	CreatedAt time.Time
}