package services

import (
	"fmt"
	"log"
	"strings"

	"idoctor-bot/app/models"

	"gorm.io/gorm"
)

// MasterService предоставляет методы для управления мастерами
type MasterService struct {
	db *gorm.DB
}

// NewMasterService создает новый экземпляр MasterService
func NewMasterService(db *gorm.DB) *MasterService {
	return &MasterService{db: db}
}

// GetAllMasters возвращает список всех мастеров
func (s *MasterService) GetAllMasters() ([]models.User, error) {
	var masters []models.User
	err := s.db.Where("role = ? AND deleted_at IS NULL", models.UserRoleMaster).
		Order("is_active DESC, name ASC").
		Find(&masters).Error
	
	if err != nil {
		return nil, fmt.Errorf("ошибка получения мастеров: %v", err)
	}
	
	return masters, nil
}

// GetActiveMasters возвращает список активных мастеров
func (s *MasterService) GetActiveMasters() ([]models.User, error) {
	var masters []models.User
	err := s.db.Where("role = ? AND is_active = ? AND deleted_at IS NULL", models.UserRoleMaster, true).
		Order("name ASC").
		Find(&masters).Error
	
	if err != nil {
		return nil, fmt.Errorf("ошибка получения активных мастеров: %v", err)
	}
	
	return masters, nil
}

// GetMasterByID возвращает мастера по ID
func (s *MasterService) GetMasterByID(id uint) (*models.User, error) {
	var master models.User
	err := s.db.Where("id = ? AND role = ? AND deleted_at IS NULL", id, models.UserRoleMaster).
		First(&master).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("мастер не найден")
		}
		return nil, fmt.Errorf("ошибка получения мастера: %v", err)
	}
	
	return &master, nil
}

// GetMasterWithStats возвращает мастера со статистикой
func (s *MasterService) GetMasterWithStats(id uint) (*MasterProfileInfo, error) {
	master, err := s.GetMasterByID(id)
	if err != nil {
		return nil, err
	}
	
	// Получаем статистику мастера
	var deviceCount, completedCount, activeCount int64
	var totalRevenue float64
	
	// Общее количество устройств
	s.db.Model(&models.Device{}).Where("master_id = ?", id).Count(&deviceCount)
	
	// Завершенные заказы
	s.db.Model(&models.Device{}).Where("master_id = ? AND status = ?", id, models.DeviceStatusCompleted).Count(&completedCount)
	
	// Активные заказы
	s.db.Model(&models.Device{}).Where("master_id = ? AND status IN ?", id, 
		[]models.DeviceStatus{models.DeviceStatusReceived, models.DeviceStatusInProgress, models.DeviceStatusWaitingParts}).Count(&activeCount)
	
	// Общая выручка
	s.db.Model(&models.Device{}).Where("master_id = ? AND status = ?", id, models.DeviceStatusCompleted).
		Select("COALESCE(SUM(total_cost), 0)").Scan(&totalRevenue)
	
	info := &MasterProfileInfo{
		User:            *master,
		TotalOrders:     int(deviceCount),
		CompletedOrders: int(completedCount),
		ActiveOrders:    int(activeCount),
		TotalRevenue:    totalRevenue,
	}
	
	// Рассчитываем рейтинг
	if info.TotalOrders > 0 {
		completionRate := float64(info.CompletedOrders) / float64(info.TotalOrders)
		info.Rating = completionRate * 10
	}
	
	return info, nil
}

// ToggleMasterStatus переключает активность мастера
func (s *MasterService) ToggleMasterStatus(id uint) error {
	var master models.User
	err := s.db.Where("id = ? AND role = ?", id, models.UserRoleMaster).First(&master).Error
	if err != nil {
		return fmt.Errorf("мастер не найден: %v", err)
	}
	
	master.IsActive = !master.IsActive
	err = s.db.Save(&master).Error
	if err != nil {
		return fmt.Errorf("ошибка изменения статуса мастера: %v", err)
	}
	
	log.Printf("Master %d status changed to %t", master.ID, master.IsActive)
	return nil
}

// AssignMasterToDevice назначает мастера на устройство
func (s *MasterService) AssignMasterToDevice(deviceID, masterID uint) error {
	// Проверяем, что мастер существует и активен
	var master models.User
	err := s.db.Where("id = ? AND role = ? AND is_active = ?", masterID, models.UserRoleMaster, true).
		First(&master).Error
	if err != nil {
		return fmt.Errorf("активный мастер не найден: %v", err)
	}
	
	// Обновляем устройство
	err = s.db.Model(&models.Device{}).Where("id = ?", deviceID).
		Update("master_id", masterID).Error
	if err != nil {
		return fmt.Errorf("ошибка назначения мастера: %v", err)
	}
	
	log.Printf("Master %d assigned to device %d", masterID, deviceID)
	return nil
}

// GetMasterDevices возвращает устройства мастера с фильтрацией
func (s *MasterService) GetMasterDevices(masterID uint, status models.DeviceStatus) ([]models.Device, error) {
	query := s.db.Where("master_id = ?", masterID).Preload("Customer")
	
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	var devices []models.Device
	err := query.Order("created_at DESC").Find(&devices).Error
	if err != nil {
		return nil, fmt.Errorf("ошибка получения устройств мастера: %v", err)
	}
	
	return devices, nil
}

// FormatMastersList форматирует список мастеров для отображения в Telegram
func (s *MasterService) FormatMastersList(masters []models.User, lang string) string {
	var result strings.Builder
	
	result.WriteString("👥 ")
	switch lang {
	case "ru":
		result.WriteString("**Список мастеров**\n\n")
	case "uz":
		result.WriteString("**Ustalar ro'yxati**\n\n")
	default:
		result.WriteString("**Masters List**\n\n")
	}
	
	if len(masters) == 0 {
		switch lang {
		case "ru":
			result.WriteString("Мастера не найдены")
		case "uz":
			result.WriteString("Ustalar topilmadi")
		default:
			result.WriteString("No masters found")
		}
		return result.String()
	}
	
	for i, master := range masters {
		// Получаем краткую статистику
		var deviceCount int64
		s.db.Model(&models.Device{}).Where("master_id = ?", master.ID).Count(&deviceCount)
		
		var activeCount int64
		s.db.Model(&models.Device{}).Where("master_id = ? AND status IN ?", master.ID,
			[]models.DeviceStatus{models.DeviceStatusReceived, models.DeviceStatusInProgress, models.DeviceStatusWaitingParts}).Count(&activeCount)
		
		statusIcon := "✅"
		statusText := "Активен"
		if !master.IsActive {
			statusIcon = "❌"
			statusText = "Неактивен"
		}
		
		switch lang {
		case "uz":
			if !master.IsActive {
				statusText = "Faol emas"
			} else {
				statusText = "Faol"
			}
		case "en":
			if !master.IsActive {
				statusText = "Inactive"
			} else {
				statusText = "Active"
			}
		}
		
		result.WriteString(fmt.Sprintf("**%d. %s**\n", i+1, master.FullName()))
		result.WriteString(fmt.Sprintf("%s %s\n", statusIcon, statusText))
		result.WriteString(fmt.Sprintf("📊 Заказов: %d (активных: %d)\n", deviceCount, activeCount))
		result.WriteString(fmt.Sprintf("🆔 ID: %d\n\n", master.ID))
	}
	
	return result.String()
}

// FormatMasterProfile форматирует профиль мастера для отображения
func (s *MasterService) FormatMasterProfile(info *MasterProfileInfo, lang string) string {
	var result strings.Builder
	
	result.WriteString("👨‍🔧 ")
	switch lang {
	case "ru":
		result.WriteString("**Профиль мастера**\n\n")
	case "uz":
		result.WriteString("**Usta profili**\n\n")
	default:
		result.WriteString("**Master Profile**\n\n")
	}
	
	result.WriteString(fmt.Sprintf("**%s**\n", info.User.FullName()))
	
	statusIcon := "✅"
	statusText := "Активен"
	if !info.User.IsActive {
		statusIcon = "❌"
		statusText = "Неактивен"
	}
	
	switch lang {
	case "uz":
		if !info.User.IsActive {
			statusText = "Faol emas"
		} else {
			statusText = "Faol"
		}
	case "en":
		if !info.User.IsActive {
			statusText = "Inactive"
		} else {
			statusText = "Active"
		}
	}
	
	result.WriteString(fmt.Sprintf("%s Статус: %s\n", statusIcon, statusText))
	result.WriteString(fmt.Sprintf("🆔 ID: %d\n", info.User.ID))
	result.WriteString(fmt.Sprintf("💬 Telegram ID: %d\n", info.User.TelegramID))
	if info.User.Username != nil {
		result.WriteString(fmt.Sprintf("👤 Username: @%s\n", *info.User.Username))
	}
	result.WriteString("\n")
	
	// Статистика
	result.WriteString("📊 ")
	switch lang {
	case "ru":
		result.WriteString("**Статистика:**\n")
	case "uz":
		result.WriteString("**Statistika:**\n")
	default:
		result.WriteString("**Statistics:**\n")
	}
	
	result.WriteString(fmt.Sprintf("• Всего заказов: %d\n", info.TotalOrders))
	result.WriteString(fmt.Sprintf("• Завершено: %d\n", info.CompletedOrders))
	result.WriteString(fmt.Sprintf("• В работе: %d\n", info.ActiveOrders))
	result.WriteString(fmt.Sprintf("• Выручка: %.2f сум\n", info.TotalRevenue))
	
	if info.Rating > 0 {
		result.WriteString(fmt.Sprintf("• Рейтинг: %.1f/10\n", info.Rating))
	}
	
	return result.String()
}

// MasterProfileInfo содержит информацию о профиле мастера
type MasterProfileInfo struct {
	User            models.User `json:"user"`
	TotalOrders     int         `json:"total_orders"`
	CompletedOrders int         `json:"completed_orders"`
	ActiveOrders    int         `json:"active_orders"`
	TotalRevenue    float64     `json:"total_revenue"`
	Rating          float64     `json:"rating"`
}