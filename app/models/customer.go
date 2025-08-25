package models

import (
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Phone     string    `gorm:"size:20;not null;uniqueIndex" json:"phone"`
	Address   string    `gorm:"size:255" json:"address"`
	Email     string    `gorm:"size:100" json:"email"`
	Notes     string    `gorm:"type:text" json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Связи
	Devices []Device `gorm:"foreignKey:CustomerID" json:"devices,omitempty"`
}

func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Customer) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Customer) TableName() string {
	return "customers"
}