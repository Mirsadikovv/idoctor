package models

import (
	"gorm.io/gorm"
	"time"
)

type UserStateType string

const (
	StateIdle         UserStateType = "idle"
	StateSettingPrice UserStateType = "setting_price"
	// Состояния для подачи заявки на ремонт (администраторы)
	StateWaitingCustomerName     UserStateType = "waiting_customer_name"
	StateWaitingCustomerPhone    UserStateType = "waiting_customer_phone"
	StateWaitingDeviceBrand      UserStateType = "waiting_device_brand"
	StateWaitingDeviceModel      UserStateType = "waiting_device_model"
	StateWaitingDeviceIssue      UserStateType = "waiting_device_issue"
	StateConfirmingOrder         UserStateType = "confirming_order"
	StateWaitingMasterTelegramID UserStateType = "waiting_master_telegram_id"
	// Состояния для создания заказов клиентами
	StateClientWaitingDeviceType   UserStateType = "client_waiting_device_type"
	StateClientWaitingDeviceBrand  UserStateType = "client_waiting_device_brand"
	StateClientWaitingDeviceModel  UserStateType = "client_waiting_device_model"
	StateClientWaitingProblem      UserStateType = "client_waiting_problem"
	StateClientWaitingContactName  UserStateType = "client_waiting_contact_name"
	StateClientWaitingContactPhone UserStateType = "client_waiting_contact_phone"
	StateClientConfirmingOrder     UserStateType = "client_confirming_order"
)

type UserState struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	TelegramID int64          `json:"telegram_id" gorm:"index;not null"`
	State      UserStateType  `json:"state" gorm:"size:50;default:'idle'"`
	Data       string         `json:"data" gorm:"type:text"` // JSON данные для состояния
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (us *UserState) TableName() string {
	return "user_states"
}

// OrderData структура для хранения данных заказа во время создания (администраторы)
type OrderData struct {
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	DeviceBrand   string `json:"device_brand"`
	DeviceModel   string `json:"device_model"`
	DeviceIssue   string `json:"device_issue"`
	MasterID      uint   `json:"master_id,omitempty"`
}

// ClientOrderData структура для хранения данных заказа клиента во время создания
type ClientOrderData struct {
	DeviceType    string `json:"device_type"`
	DeviceBrand   string `json:"device_brand"`
	DeviceModel   string `json:"device_model"`
	Problem       string `json:"problem"`
	ContactName   string `json:"contact_name"`
	ContactPhone  string `json:"contact_phone"`
	SerialNumber  string `json:"serial_number,omitempty"`
}
