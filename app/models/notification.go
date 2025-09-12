package models

import (
	"time"

	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationTypeDeviceReady     NotificationType = "device_ready"     // Устройство готово
	NotificationTypeDeviceOverdue   NotificationType = "device_overdue"   // Устройство просрочено
	NotificationTypeLowStock        NotificationType = "low_stock"        // Мало запчастей
	NotificationTypePaymentReceived NotificationType = "payment_received" // Получена оплата
	NotificationTypeDeviceReceived  NotificationType = "device_received"  // Устройство принято
	NotificationTypeNewOrder        NotificationType = "new_order"        // Новый заказ
)

type Notification struct {
	ID        uint                   `gorm:"primaryKey" json:"id"`
	UserID    *uint                  `json:"user_id"` // null для всех пользователей
	Type      string                 `gorm:"size:50;not null" json:"type"`
	Title     string                 `gorm:"size:200;not null" json:"title"`
	Message   string                 `gorm:"type:text;not null" json:"message"`
	Data      map[string]interface{} `gorm:"serializer:json" json:"data"` // JSON данные
	IsRead    bool                   `gorm:"default:false" json:"is_read"`
	IsSent    bool                   `gorm:"default:false" json:"is_sent"`
	SentAt    *time.Time             `json:"sent_at"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`

	// Связи
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()
	return nil
}

func (n *Notification) BeforeUpdate(tx *gorm.DB) error {
	n.UpdatedAt = time.Now()
	return nil
}

func (n *Notification) MarkAsRead() {
	n.IsRead = true
	n.UpdatedAt = time.Now()
}

func (n *Notification) MarkAsSent() {
	n.IsSent = true
	now := time.Now()
	n.SentAt = &now
	n.UpdatedAt = time.Now()
}

func (n *Notification) TableName() string {
	return "notifications"
}
