package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DeviceStatus string

const (
	DeviceStatusReceived     DeviceStatus = "received"
	DeviceStatusInProgress   DeviceStatus = "inProgress"
	DeviceStatusWaitingParts DeviceStatus = "waitingParts"
	DeviceStatusReady        DeviceStatus = "ready"
	DeviceStatusCompleted    DeviceStatus = "completed"
	DeviceStatusCancelled    DeviceStatus = "cancelled"
)

func (d DeviceStatus) String() string {
	return string(d)
}

func (d DeviceStatus) IsValid() bool {
	switch d {
	case DeviceStatusReceived, DeviceStatusInProgress, DeviceStatusWaitingParts,
		DeviceStatusReady, DeviceStatusCompleted, DeviceStatusCancelled:
		return true
	}
	return false
}

func (d DeviceStatus) Text() string {
	switch d {
	case DeviceStatusReceived:
		return "Принят"
	case DeviceStatusInProgress:
		return "В работе"
	case DeviceStatusWaitingParts:
		return "Ожидание запчастей"
	case DeviceStatusReady:
		return "Готов"
	case DeviceStatusCompleted:
		return "Выдан"
	case DeviceStatusCancelled:
		return "Отменен"
	default:
		return string(d)
	}
}

type Device struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	Code         string       `gorm:"size:20;uniqueIndex;not null" json:"code"`
	CustomerID   uint         `gorm:"not null" json:"customer_id"`
	MasterID     *uint        `json:"master_id"`
	DeviceType   string       `gorm:"size:100;not null" json:"device_type"`
	Brand        string       `gorm:"size:100" json:"brand"`
	Model        string       `gorm:"size:100" json:"model"`
	SerialNumber string       `gorm:"size:100" json:"serial_number"`
	Problem      string       `gorm:"type:text;not null" json:"problem"`
	Status       DeviceStatus `gorm:"type:varchar(20);default:'received'" json:"status"`
	Diagnosis    string       `gorm:"type:text" json:"diagnosis"`
	RepairCost   float64      `gorm:"type:decimal(10,2);default:0" json:"repair_cost"`
	PartsCost    float64      `gorm:"type:decimal(10,2);default:0" json:"parts_cost"`
	TotalCost    float64      `gorm:"type:decimal(10,2);default:0" json:"total_cost"`
	IsPaid       bool         `gorm:"default:false" json:"is_paid"`
	Notes        string       `gorm:"type:text" json:"notes"`
	ReceivedAt   time.Time    `json:"received_at"`
	DeadlineAt   *time.Time   `json:"deadline_at"`
	CompletedAt  *time.Time   `json:"completed_at"`
	DeliveredAt  *time.Time   `json:"delivered_at"`
	WarrantyDays int          `gorm:"default:0" json:"warranty_days"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`

	// Связи
	Customer *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Master   *User     `gorm:"foreignKey:MasterID" json:"master,omitempty"`
	Parts    []Part    `gorm:"many2many:device_parts;" json:"parts,omitempty"`
}

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	if d.ReceivedAt.IsZero() {
		d.ReceivedAt = time.Now()
	}
	d.CreatedAt = time.Now()
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) IsOverdue() bool {
	if d.DeadlineAt == nil || d.Status == DeviceStatusCompleted || d.Status == DeviceStatusCancelled {
		return false
	}
	return time.Now().After(*d.DeadlineAt)
}

func (d *Device) CanBeUpdatedBy(userRole UserRole, userID uint) bool {
	if userRole == UserRoleAdmin {
		return true
	}
	if userRole == UserRoleMaster && d.MasterID != nil && *d.MasterID == userID {
		return true
	}
	return false
}

func (d *Device) FormatDetails() string {
	details := ""
	if d.Code != "" {
		details += "🆔 Код: " + d.Code + "\n"
	}
	if d.Customer != nil {
		details += "👤 Клиент: " + d.Customer.Name + "\n"
		details += "📞 Телефон: " + d.Customer.Phone + "\n"
	}
	if d.DeviceType != "" {
		details += "📱 Тип: " + d.DeviceType + "\n"
	}
	if d.Brand != "" || d.Model != "" {
		details += "🏷️ Модель: " + d.Brand + " " + d.Model + "\n"
	}
	if d.Problem != "" {
		details += "❗ Проблема: " + d.Problem + "\n"
	}
	details += "📊 Статус: " + d.Status.Text() + "\n"
	if d.Master != nil {
		details += "🔧 Мастер: " + d.Master.FullName() + "\n"
	}
	if d.RepairCost > 0 {
		details += "💰 Цена ремонта: " + formatMoney(d.RepairCost) + "\n"
	}
	if d.PartsCost > 0 {
		details += "🔩 Цена запчастей: " + formatMoney(d.PartsCost) + "\n"
	}
	if d.DeadlineAt != nil {
		details += "⏰ Срок: " + d.DeadlineAt.Format("02.01.2006 15:04") + "\n"
	}
	return details
}

// CalculateTotalCost вычисляет общую стоимость как сумму ремонта и запчастей
func (d *Device) CalculateTotalCost() {
	d.TotalCost = d.RepairCost + d.PartsCost
}

// SetRepairCost устанавливает стоимость ремонта и пересчитывает общую стоимость
func (d *Device) SetRepairCost(cost float64) {
	if cost < 0 {
		cost = 0
	}
	d.RepairCost = cost
	d.CalculateTotalCost()
}

// SetPartsCost устанавливает стоимость запчастей и пересчитывает общую стоимость
func (d *Device) SetPartsCost(cost float64) {
	if cost < 0 {
		cost = 0
	}
	d.PartsCost = cost
	d.CalculateTotalCost()
}

// SetPaid устанавливает статус оплаты
func (d *Device) SetPaid(paid bool) {
	d.IsPaid = paid
}

// GetProfitability возвращает рентабельность заказа (только для завершенных и оплаченных)
func (d *Device) GetProfitability() float64 {
	if d.Status != DeviceStatusCompleted || !d.IsPaid || d.TotalCost <= 0 {
		return 0
	}
	// Простая формула рентабельности: стоимость работ минус стоимость запчастей
	return d.RepairCost
}

// IsReadyForPricing проверяет, можно ли установить цену (статус позволяет)
func (d *Device) IsReadyForPricing() bool {
	return d.Status == DeviceStatusInProgress || d.Status == DeviceStatusWaitingParts || d.Status == DeviceStatusReady
}

// FormatPricing возвращает форматированную информацию о ценах
func (d *Device) FormatPricing() string {
	result := ""
	if d.RepairCost > 0 {
		result += fmt.Sprintf("💰 Ремонт: %s\n", formatMoney(d.RepairCost))
	}
	if d.PartsCost > 0 {
		result += fmt.Sprintf("🔩 Запчасти: %s\n", formatMoney(d.PartsCost))
	}
	if d.TotalCost > 0 {
		result += fmt.Sprintf("💸 Итого: %s\n", formatMoney(d.TotalCost))
		if d.IsPaid {
			result += "✅ Оплачено\n"
		} else {
			result += "❌ Не оплачено\n"
		}
	} else {
		result += "💰 Цена не установлена\n"
	}
	return result
}

func (d *Device) TableName() string {
	return "devices"
}

// Helper function to format money
func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f сум", amount)
}
