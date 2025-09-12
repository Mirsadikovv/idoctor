package services

import (
	"fmt"
	"strconv"
	"strings"

	"idoctor-bot/app/models"

	"gorm.io/gorm"
)

// PricingService предоставляет методы для управления ценами ремонта
type PricingService struct {
	db *gorm.DB
}

// NewPricingService создает новый экземпляр PricingService
func NewPricingService(db *gorm.DB) *PricingService {
	return &PricingService{db: db}
}

// SetRepairPrice устанавливает цену ремонта для устройства
func (s *PricingService) SetRepairPrice(deviceID uint, price float64, userID uint, userRole models.UserRole) error {
	var device models.Device
	if err := s.db.First(&device, deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("устройство не найдено")
		}
		return fmt.Errorf("ошибка получения устройства: %v", err)
	}

	// Проверяем права доступа
	if !device.CanBeUpdatedBy(userRole, userID) {
		return fmt.Errorf("нет прав для изменения цены этого заказа")
	}

	// Проверяем, можно ли установить цену (статус позволяет)
	if !device.IsReadyForPricing() {
		return fmt.Errorf("нельзя установить цену для заказа в статусе '%s'", device.Status.Text())
	}

	device.SetRepairCost(price)

	if err := s.db.Save(&device).Error; err != nil {
		return fmt.Errorf("ошибка сохранения цены ремонта: %v", err)
	}

	return nil
}

// SetPartsPrice устанавливает цену запчастей для устройства
func (s *PricingService) SetPartsPrice(deviceID uint, price float64, userID uint, userRole models.UserRole) error {
	var device models.Device
	if err := s.db.First(&device, deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("устройство не найдено")
		}
		return fmt.Errorf("ошибка получения устройства: %v", err)
	}

	// Проверяем права доступа
	if !device.CanBeUpdatedBy(userRole, userID) {
		return fmt.Errorf("нет прав для изменения цены этого заказа")
	}

	// Проверяем, можно ли установить цену
	if !device.IsReadyForPricing() {
		return fmt.Errorf("нельзя установить цену для заказа в статусе '%s'", device.Status.Text())
	}

	device.SetPartsCost(price)

	if err := s.db.Save(&device).Error; err != nil {
		return fmt.Errorf("ошибка сохранения цены запчастей: %v", err)
	}

	return nil
}

// SetPaidStatus устанавливает статус оплаты заказа
func (s *PricingService) SetPaidStatus(deviceID uint, paid bool, userID uint, userRole models.UserRole) error {
	var device models.Device
	if err := s.db.Preload("Customer").First(&device, deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("устройство не найдено")
		}
		return fmt.Errorf("ошибка получения устройства: %v", err)
	}

	// Проверяем права доступа
	if !device.CanBeUpdatedBy(userRole, userID) {
		return fmt.Errorf("нет прав для изменения статуса оплаты этого заказа")
	}

	// Можно изменить статус оплаты только если есть цена
	if device.TotalCost <= 0 {
		return fmt.Errorf("нельзя изменить статус оплаты без установленной цены")
	}

	device.SetPaid(paid)

	if err := s.db.Save(&device).Error; err != nil {
		return fmt.Errorf("ошибка сохранения статуса оплаты: %v", err)
	}

	return nil
}

// GetDeviceWithPricing возвращает устройство с информацией о ценах
func (s *PricingService) GetDeviceWithPricing(deviceID uint) (*models.Device, error) {
	var device models.Device
	if err := s.db.Preload("Customer").Preload("Master").First(&device, deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("устройство не найдено")
		}
		return nil, fmt.Errorf("ошибка получения устройства: %v", err)
	}

	return &device, nil
}

// ParsePriceFromText парсит цену из текстового сообщения
func (s *PricingService) ParsePriceFromText(text string) (float64, error) {
	// Очищаем текст от лишних символов и приводим к нижнему регистру
	cleanText := strings.ToLower(strings.TrimSpace(text))
	cleanText = strings.ReplaceAll(cleanText, " ", "")
	cleanText = strings.ReplaceAll(cleanText, ",", ".")
	
	// Убираем возможные валютные обозначения
	cleanText = strings.ReplaceAll(cleanText, "сум", "")
	cleanText = strings.ReplaceAll(cleanText, "sum", "")
	cleanText = strings.ReplaceAll(cleanText, "$", "")
	cleanText = strings.ReplaceAll(cleanText, "₽", "")
	
	// Парсим число
	price, err := strconv.ParseFloat(cleanText, 64)
	if err != nil {
		return 0, fmt.Errorf("неверный формат цены. Введите число (например: 50000 или 50000.50)")
	}

	if price < 0 {
		return 0, fmt.Errorf("цена не может быть отрицательной")
	}

	if price > 999999999 {
		return 0, fmt.Errorf("цена слишком большая")
	}

	return price, nil
}

// GetUnpaidDevices возвращает список неоплаченных устройств
func (s *PricingService) GetUnpaidDevices(userID uint, userRole models.UserRole) ([]models.Device, error) {
	var devices []models.Device
	
	query := s.db.Preload("Customer").Preload("Master").
		Where("total_cost > ? AND is_paid = ?", 0, false)

	// Если это мастер, показываем только его заказы
	if userRole == models.UserRoleMaster {
		query = query.Where("master_id = ?", userID)
	}

	if err := query.Order("created_at DESC").Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("ошибка получения неоплаченных заказов: %v", err)
	}

	return devices, nil
}

// GetDevicesForPricing возвращает список устройств, для которых можно установить цену
func (s *PricingService) GetDevicesForPricing(userID uint, userRole models.UserRole) ([]models.Device, error) {
	var devices []models.Device
	
	query := s.db.Preload("Customer").Preload("Master").
		Where("status IN (?)", []string{
			string(models.DeviceStatusInProgress),
			string(models.DeviceStatusWaitingParts),
			string(models.DeviceStatusReady),
		})

	// Если это мастер, показываем только его заказы
	if userRole == models.UserRoleMaster {
		query = query.Where("master_id = ?", userID)
	}

	if err := query.Order("created_at DESC").Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("ошибка получения заказов для установки цен: %v", err)
	}

	return devices, nil
}

// FormatPricingInfo форматирует информацию о ценах для отображения
func (s *PricingService) FormatPricingInfo(device *models.Device) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("🆔 **Заказ %s**\n", device.Code))
	if device.Customer != nil {
		result.WriteString(fmt.Sprintf("👤 %s\n", device.Customer.Name))
	}
	result.WriteString(fmt.Sprintf("📱 %s %s\n", device.Brand, device.Model))
	result.WriteString(fmt.Sprintf("📊 %s\n\n", device.Status.Text()))

	result.WriteString(device.FormatPricing())

	return result.String()
}

// CalculateRevenueStatistics вычисляет статистику доходов
func (s *PricingService) CalculateRevenueStatistics(period models.StatisticsPeriod) (*RevenueStats, error) {
	startDate, endDate := period.GetPeriodDates()

	var stats RevenueStats
	
	// Общая статистика
	var totalRevenue, paidRevenue, unpaidRevenue float64
	var totalOrders, paidOrders, unpaidOrders int64

	// Всего заказов за период
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ?", 0).
		Count(&totalOrders)

	// Общая сумма всех заказов
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ?", 0).
		Select("COALESCE(SUM(total_cost), 0)").
		Row().Scan(&totalRevenue)

	// Оплаченные заказы
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, true).
		Count(&paidOrders)

	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, true).
		Select("COALESCE(SUM(total_cost), 0)").
		Row().Scan(&paidRevenue)

	// Неоплаченные заказы
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, false).
		Count(&unpaidOrders)

	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, false).
		Select("COALESCE(SUM(total_cost), 0)").
		Row().Scan(&unpaidRevenue)

	stats.TotalOrders = int(totalOrders)
	stats.PaidOrders = int(paidOrders)
	stats.UnpaidOrders = int(unpaidOrders)
	stats.TotalRevenue = totalRevenue
	stats.PaidRevenue = paidRevenue
	stats.UnpaidRevenue = unpaidRevenue

	if paidOrders > 0 {
		stats.AverageOrderValue = paidRevenue / float64(paidOrders)
	}

	return &stats, nil
}

// RevenueStats содержит статистику доходов
type RevenueStats struct {
	TotalOrders       int     `json:"total_orders"`
	PaidOrders        int     `json:"paid_orders"`
	UnpaidOrders      int     `json:"unpaid_orders"`
	TotalRevenue      float64 `json:"total_revenue"`
	PaidRevenue       float64 `json:"paid_revenue"`
	UnpaidRevenue     float64 `json:"unpaid_revenue"`
	AverageOrderValue float64 `json:"average_order_value"`
}

// FormatRevenueStats форматирует статистику доходов для отображения
func (s *PricingService) FormatRevenueStats(stats *RevenueStats, period string) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("💰 **Финансовая статистика: %s**\n\n", period))

	result.WriteString("📊 **Заказы:**\n")
	result.WriteString(fmt.Sprintf("• Всего: %d\n", stats.TotalOrders))
	result.WriteString(fmt.Sprintf("• Оплачено: %d\n", stats.PaidOrders))
	result.WriteString(fmt.Sprintf("• Не оплачено: %d\n\n", stats.UnpaidOrders))

	result.WriteString("💸 **Доходы:**\n")
	result.WriteString(fmt.Sprintf("• Общая сумма: %s\n", formatMoney(stats.TotalRevenue)))
	result.WriteString(fmt.Sprintf("• Получено: %s\n", formatMoney(stats.PaidRevenue)))
	result.WriteString(fmt.Sprintf("• К получению: %s\n", formatMoney(stats.UnpaidRevenue)))

	if stats.AverageOrderValue > 0 {
		result.WriteString(fmt.Sprintf("• Средний чек: %s\n", formatMoney(stats.AverageOrderValue)))
	}

	if stats.TotalOrders > 0 {
		paymentRate := float64(stats.PaidOrders) / float64(stats.TotalOrders) * 100
		result.WriteString(fmt.Sprintf("• Процент оплат: %.1f%%\n", paymentRate))
	}

	return result.String()
}

// formatMoney форматирует сумму для отображения (локальная функция)
func formatMoney(amount float64) string {
	return fmt.Sprintf("%.2f сум", amount)
}