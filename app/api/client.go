package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIClient интерфейс для работы с основной системой
type APIClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewAPIClient создает новый клиент для API основной системы
func NewAPIClient(baseURL, apiKey string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Device представляет устройство в системе
type Device struct {
	ID           uint     `json:"id"`
	CustomerID   uint     `json:"customer_id"`
	MasterID     *uint    `json:"master_id"`
	Model        string   `json:"model"`
	Brand        string   `json:"brand"`
	SerialNumber string   `json:"serial_number"`
	Issue        string   `json:"issue"`
	Status       string   `json:"status"`
	Price        *float64 `json:"price"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	// Связи
	Customer *Customer `json:"customer,omitempty"`
	Master   *User     `json:"master,omitempty"`
}

// Customer представляет клиента
type Customer struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
	Address     string `json:"address"`
}

// User представляет пользователя системы
type User struct {
	ID         uint   `json:"id"`
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	IsActive   bool   `json:"is_active"`
}

// GetDevices получает список устройств
func (c *APIClient) GetDevices() ([]Device, error) {
	// Пока API сервер не готов, возвращаем мок данные
	if c.baseURL == "http://localhost:8080" {
		return c.getMockDevices(), nil
	}

	url := fmt.Sprintf("%s/api/v1/devices", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var devices []Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, err
	}

	return devices, nil
}

// GetDevicesByMaster получает устройства конкретного мастера
func (c *APIClient) GetDevicesByMaster(masterID uint) ([]Device, error) {
	// Пока API сервер не готов, возвращаем мок данные
	if c.baseURL == "http://localhost:8080" {
		allDevices := c.getMockDevices()
		var masterDevices []Device
		for _, device := range allDevices {
			if device.MasterID != nil && *device.MasterID == masterID {
				masterDevices = append(masterDevices, device)
			}
		}
		return masterDevices, nil
	}

	url := fmt.Sprintf("%s/api/v1/devices?master_id=%d", c.baseURL, masterID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var devices []Device
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, err
	}

	return devices, nil
}

// UpdateDeviceStatus обновляет статус устройства
func (c *APIClient) UpdateDeviceStatus(deviceID uint, status string) error {
	// Пока API сервер не готов, просто логируем
	if c.baseURL == "http://localhost:8080" {
		// Мок - просто возвращаем успех
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/devices/%d/status", c.baseURL, deviceID)

	payload := map[string]string{"status": status}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// getMockDevices возвращает тестовые данные
func (c *APIClient) getMockDevices() []Device {
	price1 := 150000.0
	price2 := 75000.0
	masterID1 := uint(1)
	masterID2 := uint(2)

	return []Device{
		{
			ID:           1,
			CustomerID:   1,
			MasterID:     &masterID1,
			Model:        "iPhone 12",
			Brand:        "Apple",
			SerialNumber: "ABC123456789",
			Issue:        "Не включается, попадала в воду",
			Status:       "inProgress",
			Price:        &price1,
			CreatedAt:    "2024-01-15T10:00:00Z",
			UpdatedAt:    "2024-01-15T10:00:00Z",
			Customer: &Customer{
				ID:          1,
				Name:        "Иван Иванов",
				PhoneNumber: "+998901234567",
				Email:       "ivan@example.com",
			},
			Master: &User{
				ID:         1,
				TelegramID: 833391285,
				Name:       "Мастер Иван",
				Role:       "master",
				IsActive:   true,
			},
		},
		{
			ID:           2,
			CustomerID:   2,
			MasterID:     &masterID2,
			Model:        "Galaxy S21",
			Brand:        "Samsung",
			SerialNumber: "DEF987654321",
			Issue:        "Разбитый экран",
			Status:       "ready",
			Price:        &price2,
			CreatedAt:    "2024-01-16T09:30:00Z",
			UpdatedAt:    "2024-01-16T09:30:00Z",
			Customer: &Customer{
				ID:          2,
				Name:        "Мария Петрова",
				PhoneNumber: "+998907654321",
				Email:       "maria@example.com",
			},
			Master: &User{
				ID:         2,
				TelegramID: 7233051530,
				Name:       "Мастер Петр",
				Role:       "master",
				IsActive:   true,
			},
		},
		{
			ID:         3,
			CustomerID: 3,
			MasterID:   nil,
			Model:      "iPhone 13",
			Brand:      "Apple",
			Issue:      "Батарея быстро разряжается",
			Status:     "received",
			Price:      nil,
			CreatedAt:  "2024-01-17T14:20:00Z",
			UpdatedAt:  "2024-01-17T14:20:00Z",
			Customer: &Customer{
				ID:          3,
				Name:        "Алексей Сидоров",
				PhoneNumber: "+998903456789",
			},
		},
	}
}

// CreateDevice создает новое устройство
func (c *APIClient) CreateDevice(device Device) (*Device, error) {
	url := fmt.Sprintf("%s/api/v1/devices", c.baseURL)

	jsonPayload, err := json.Marshal(device)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var createdDevice Device
	if err := json.NewDecoder(resp.Body).Decode(&createdDevice); err != nil {
		return nil, err
	}

	return &createdDevice, nil
}

// GetUserByTelegramID получает пользователя по Telegram ID
func (c *APIClient) GetUserByTelegramID(telegramID int64) (*User, error) {
	url := fmt.Sprintf("%s/api/v1/users?telegram_id=%d", c.baseURL, telegramID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateCustomer создает нового клиента
func (c *APIClient) CreateCustomer(customer Customer) (*Customer, error) {
	// Мок для тестирования
	if c.baseURL == "http://localhost:8080" {
		// Присваиваем ID для мок-данных
		customer.ID = uint(time.Now().Unix() % 10000)
		return &customer, nil
	}

	url := fmt.Sprintf("%s/api/v1/customers", c.baseURL)

	jsonPayload, err := json.Marshal(customer)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var createdCustomer Customer
	if err := json.NewDecoder(resp.Body).Decode(&createdCustomer); err != nil {
		return nil, err
	}

	return &createdCustomer, nil
}

// UpdateDevicePrice обновляет цену устройства
func (c *APIClient) UpdateDevicePrice(deviceID uint, price float64) error {
	// Мок для тестирования
	if c.baseURL == "http://localhost:8080" {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/devices/%d/price", c.baseURL, deviceID)

	payload := map[string]float64{"price": price}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// AssignMaster назначает мастера для устройства
func (c *APIClient) AssignMaster(deviceID uint, masterID uint) error {
	// Мок для тестирования
	if c.baseURL == "http://localhost:8080" {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/devices/%d/master", c.baseURL, deviceID)

	payload := map[string]uint{"master_id": masterID}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// SendNotification отправляет уведомление через основную систему
func (c *APIClient) SendNotification(userID *uint, message string) error {
	url := fmt.Sprintf("%s/api/v1/bot/send-notification", c.baseURL)

	payload := map[string]interface{}{
		"message": message,
	}
	if userID != nil {
		payload["user_id"] = *userID
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
