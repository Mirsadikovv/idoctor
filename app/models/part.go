package models

import (
	"time"

	"gorm.io/gorm"
)

type Part struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:200;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	SKU         string    `gorm:"size:100;uniqueIndex" json:"sku"`
	Price       float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	Quantity    int       `gorm:"default:0" json:"quantity"`
	MinQuantity int       `gorm:"default:0" json:"min_quantity"`
	Category    string    `gorm:"size:100" json:"category"`
	Supplier    string    `gorm:"size:200" json:"supplier"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Связи
	Devices []Device `gorm:"many2many:device_parts;" json:"devices,omitempty"`
}

func (p *Part) BeforeCreate(tx *gorm.DB) error {
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Part) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Part) IsLowStock() bool {
	return p.Quantity <= p.MinQuantity
}

func (p *Part) IsInStock() bool {
	return p.Quantity > 0
}

func (p *Part) TableName() string {
	return "parts"
}
