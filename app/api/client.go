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
	ID           uint   `json:"id"`
	CustomerID   uint   `json:"customer_id"`
	MasterID     *uint  `json:"master_id"`
	Model        string `json:"model"`
	Brand        string `json:"brand"`
	SerialNumber string `json:"serial_number"`
	Issue        string `json:"issue"`
	Status       string `json:"status"`
	Price        *float64 `json:"price"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
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