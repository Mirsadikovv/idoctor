package services

import (
	"fmt"
	"log"
	"math"
	"strings"

	"idoctor-bot/app/models"

	"gorm.io/gorm"
)

// StatisticsService предоставляет методы для сбора статистики
type StatisticsService struct {
	db *gorm.DB
}

// NewStatisticsService создает новый экземпляр StatisticsService
func NewStatisticsService(db *gorm.DB) *StatisticsService {
	return &StatisticsService{db: db}
}

// GetPeriodStatistics возвращает статистику за указанный период
func (s *StatisticsService) GetPeriodStatistics(period models.StatisticsPeriod) (*models.PeriodStatistics, error) {
	startDate, endDate := period.GetPeriodDates()

	stats := &models.PeriodStatistics{
		Period:         string(period),
		StartDate:      startDate,
		EndDate:        endDate,
		OrdersByStatus: make(map[models.DeviceStatus]int),
		OrdersByMaster: make(map[uint]int),
		TopBrands:      make(map[string]int),
		ProblemTypes:   make(map[string]int),
	}

	// Получаем все заказы за период
	var devices []models.Device
	query := s.db.Where("created_at BETWEEN ? AND ?", startDate, endDate)
	if period != models.PeriodAllTime {
		query = query.Where("created_at BETWEEN ? AND ?", startDate, endDate)
	}

	if err := query.Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("ошибка получения заказов: %v", err)
	}

	stats.TotalOrders = len(devices)

	var totalRevenue float64
	var completedCount, cancelledCount int

	// Анализируем каждый заказ
	for _, device := range devices {
		// Статистика по статусам
		stats.OrdersByStatus[device.Status]++

		// Подсчет завершенных и отмененных
		if device.Status == models.DeviceStatusCompleted {
			completedCount++
			totalRevenue += device.TotalCost
		} else if device.Status == models.DeviceStatusCancelled {
			cancelledCount++
		}

		// Статистика по мастерам
		if device.MasterID != nil {
			stats.OrdersByMaster[*device.MasterID]++
		}

		// Статистика по брендам
		if device.Brand != "" {
			stats.TopBrands[device.Brand]++
		}

		// Анализ типов проблем
		problemType := s.categorizeProblem(device.Problem)
		stats.ProblemTypes[problemType]++
	}

	stats.CompletedOrders = completedCount
	stats.CancelledOrders = cancelledCount
	stats.TotalRevenue = totalRevenue

	if completedCount > 0 {
		stats.AverageOrderValue = totalRevenue / float64(completedCount)
	}

	return stats, nil
}

// GetMasterStatistics возвращает детальную статистику по мастерам
func (s *StatisticsService) GetMasterStatistics(period models.StatisticsPeriod) (map[uint]*models.MasterStatistics, error) {
	startDate, endDate := period.GetPeriodDates()

	// Получаем всех мастеров
	var masters []models.User
	if err := s.db.Where("role = ?", models.UserRoleMaster).Find(&masters).Error; err != nil {
		return nil, fmt.Errorf("ошибка получения мастеров: %v", err)
	}

	masterStats := make(map[uint]*models.MasterStatistics)

	for _, master := range masters {
		stats := &models.MasterStatistics{
			MasterID:   master.ID,
			MasterName: master.Name,
		}

		// Получаем заказы мастера за период
		var devices []models.Device
		query := s.db.Where("master_id = ?", master.ID)
		if period != models.PeriodAllTime {
			query = query.Where("created_at BETWEEN ? AND ?", startDate, endDate)
		}

		if err := query.Find(&devices).Error; err != nil {
			log.Printf("Error getting devices for master %d: %v", master.ID, err)
			continue
		}

		stats.TotalOrders = len(devices)

		var totalRevenue float64
		var completedCount int
		var totalRepairTime float64
		var repairTimeCount int

		for _, device := range devices {
			if device.Status == models.DeviceStatusCompleted {
				completedCount++
				totalRevenue += device.TotalCost

				// Подсчет времени ремонта
				if device.CompletedAt != nil {
					repairTime := device.CompletedAt.Sub(device.ReceivedAt).Hours() / 24 // в днях
					totalRepairTime += repairTime
					repairTimeCount++
				}
			}
		}

		stats.CompletedOrders = completedCount
		stats.TotalRevenue = totalRevenue

		if repairTimeCount > 0 {
			stats.AverageRepairTime = totalRepairTime / float64(repairTimeCount)
		}

		// Простая система рейтинга на основе процента завершенных заказов и скорости
		if stats.TotalOrders > 0 {
			completionRate := float64(completedCount) / float64(stats.TotalOrders)
			speedBonus := 1.0
			if stats.AverageRepairTime > 0 && stats.AverageRepairTime <= 3 { // Быстрый ремонт (до 3 дней)
				speedBonus = 1.2
			} else if stats.AverageRepairTime > 7 { // Медленный ремонт (более 7 дней)
				speedBonus = 0.8
			}
			stats.Rating = math.Round(completionRate*speedBonus*10*100) / 100 // Рейтинг от 0 до 10
		}

		masterStats[master.ID] = stats
	}

	return masterStats, nil
}

// GetOverallStatistics возвращает общую статистику системы
func (s *StatisticsService) GetOverallStatistics() (*models.OrderStatistics, error) {
	stats := &models.OrderStatistics{
		OrdersByStatus:    make(map[models.DeviceStatus]int),
		OrdersByPeriod:    make(map[string]int),
		TopBrands:         make(map[string]int),
		ProblemCategories: make(map[string]int),
	}

	// Получаем все заказы
	var devices []models.Device
	if err := s.db.Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("ошибка получения заказов: %v", err)
	}

	stats.TotalOrders = len(devices)

	var totalRevenue float64
	var totalRepairTime float64
	var repairTimeCount int

	for _, device := range devices {
		// Статистика по статусам
		stats.OrdersByStatus[device.Status]++

		// Выручка
		if device.Status == models.DeviceStatusCompleted {
			totalRevenue += device.TotalCost

			// Время ремонта
			if device.CompletedAt != nil {
				repairTime := device.CompletedAt.Sub(device.ReceivedAt).Hours() / 24
				totalRepairTime += repairTime
				repairTimeCount++
			}
		}

		// Статистика по брендам
		if device.Brand != "" {
			stats.TopBrands[device.Brand]++
		}

		// Категории проблем
		problemType := s.categorizeProblem(device.Problem)
		stats.ProblemCategories[problemType]++

		// Статистика по периодам
		periodKey := device.CreatedAt.Format("2006-01")
		stats.OrdersByPeriod[periodKey]++
	}

	stats.TotalRevenue = totalRevenue

	if repairTimeCount > 0 {
		stats.AverageRepairTime = totalRepairTime / float64(repairTimeCount)
	}

	// Получаем статистику по мастерам
	masterStats, err := s.GetMasterStatistics(models.PeriodAllTime)
	if err != nil {
		log.Printf("Error getting master statistics: %v", err)
	} else {
		stats.MasterStats = masterStats
	}

	return stats, nil
}

// categorizeProblem категоризирует проблему по ключевым словам
func (s *StatisticsService) categorizeProblem(problem string) string {
	problem = strings.ToLower(problem)

	// Категории проблем с ключевыми словами
	categories := map[string][]string{
		"Экран":       {"экран", "дисплей", "треснул", "разбит", "черный экран", "тачскрин", "сенсор"},
		"Батарея":     {"батарея", "аккумулятор", "заряд", "разряжается", "не заряжается", "быстро садится"},
		"Кнопки":      {"кнопка", "кнопки", "громкость", "включения", "домой", "назад"},
		"Камера":      {"камера", "фотоаппарат", "не фотографирует", "размытые фото"},
		"Звук":        {"звук", "динамик", "микрофон", "тихий", "не слышно", "шумы"},
		"Связь":       {"сеть", "связь", "сигнал", "интернет", "wifi", "bluetooth", "не ловит"},
		"Зарядка":     {"зарядка", "разъем", "usb", "не заряжается", "зарядное"},
		"Программное": {"зависает", "тормозит", "не включается", "перезагружается", "глючит", "обновление"},
		"Корпус":      {"корпус", "задняя крышка", "царапины", "вмятины"},
		"Вода":        {"вода", "намок", "упал в воду", "жидкость"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(problem, keyword) {
				return category
			}
		}
	}

	return "Другое"
}

// FormatStatistics форматирует статистику для отображения в Telegram
func (s *StatisticsService) FormatStatistics(stats *models.PeriodStatistics, lang string) string {
	var result strings.Builder

	// Заголовок
	result.WriteString(fmt.Sprintf("📊 **Статистика: %s**\n", stats.Period))
	result.WriteString(fmt.Sprintf("📅 %s - %s\n\n",
		stats.StartDate.Format("02.01.2006"),
		stats.EndDate.Format("02.01.2006")))

	// Основные показатели
	result.WriteString("📈 **Основные показатели:**\n")
	result.WriteString(fmt.Sprintf("• Всего заказов: %d\n", stats.TotalOrders))
	result.WriteString(fmt.Sprintf("• Завершено: %d\n", stats.CompletedOrders))
	result.WriteString(fmt.Sprintf("• Отменено: %d\n", stats.CancelledOrders))
	result.WriteString(fmt.Sprintf("• Выручка: %.2f сум\n", stats.TotalRevenue))

	if stats.AverageOrderValue > 0 {
		result.WriteString(fmt.Sprintf("• Средний чек: %.2f сум\n", stats.AverageOrderValue))
	}
	result.WriteString("\n")

	// Статистика по статусам
	if len(stats.OrdersByStatus) > 0 {
		result.WriteString("🔄 **По статусам:**\n")
		for status, count := range stats.OrdersByStatus {
			result.WriteString(fmt.Sprintf("• %s: %d\n", status.Text(), count))
		}
		result.WriteString("\n")
	}

	// Топ брендов
	if len(stats.TopBrands) > 0 {
		result.WriteString("🏷️ **Популярные бренды:**\n")
		for brand, count := range stats.TopBrands {
			result.WriteString(fmt.Sprintf("• %s: %d\n", brand, count))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// GetFinancialStatistics возвращает финансовую статистику за период
func (s *StatisticsService) GetFinancialStatistics(period models.StatisticsPeriod) (*FinancialStats, error) {
	startDate, endDate := period.GetPeriodDates()

	var stats FinancialStats
	stats.Period = string(period)

	// Всего заказов с ценами за период
	var totalWithPrices int64
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ?", 0).
		Count(&totalWithPrices)

	// Оплаченные заказы
	var paidCount int64
	var paidRevenue float64
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, true).
		Count(&paidCount)

	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, true).
		Select("COALESCE(SUM(total_cost), 0)").
		Row().Scan(&paidRevenue)

	// Неоплаченные заказы
	var unpaidCount int64
	var unpaidRevenue float64
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, false).
		Count(&unpaidCount)

	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ? AND is_paid = ?", 0, false).
		Select("COALESCE(SUM(total_cost), 0)").
		Row().Scan(&unpaidRevenue)

	// Общая выручка
	var totalRevenue float64
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("total_cost > ?", 0).
		Select("COALESCE(SUM(total_cost), 0)").
		Row().Scan(&totalRevenue)

	// Статистика по типам доходов
	var repairRevenue, partsRevenue float64
	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("is_paid = ?", true).
		Select("COALESCE(SUM(repair_cost), 0)").
		Row().Scan(&repairRevenue)

	s.db.Model(&models.Device{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("is_paid = ?", true).
		Select("COALESCE(SUM(parts_cost), 0)").
		Row().Scan(&partsRevenue)

	stats.TotalOrdersWithPrice = int(totalWithPrices)
	stats.PaidOrders = int(paidCount)
	stats.UnpaidOrders = int(unpaidCount)
	stats.TotalRevenue = totalRevenue
	stats.PaidRevenue = paidRevenue
	stats.UnpaidRevenue = unpaidRevenue
	stats.RepairRevenue = repairRevenue
	stats.PartsRevenue = partsRevenue

	// Вычисляем производные показатели
	if paidCount > 0 {
		stats.AverageOrderValue = paidRevenue / float64(paidCount)
	}

	if totalWithPrices > 0 {
		stats.PaymentRate = float64(paidCount) / float64(totalWithPrices) * 100
	}

	return &stats, nil
}

// FormatFinancialStats форматирует финансовую статистику для отображения
func (s *StatisticsService) FormatFinancialStats(stats *FinancialStats, lang string) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("💰 **Финансовая статистика: %s**\n\n", stats.Period))

	result.WriteString("📊 **Заказы:**\n")
	result.WriteString(fmt.Sprintf("• С установленной ценой: %d\n", stats.TotalOrdersWithPrice))
	result.WriteString(fmt.Sprintf("• Оплачено: %d\n", stats.PaidOrders))
	result.WriteString(fmt.Sprintf("• Не оплачено: %d\n\n", stats.UnpaidOrders))

	result.WriteString("💸 **Доходы:**\n")
	result.WriteString(fmt.Sprintf("• Общая сумма: %s\n", formatMoney(stats.TotalRevenue)))
	result.WriteString(fmt.Sprintf("• Получено: %s\n", formatMoney(stats.PaidRevenue)))
	result.WriteString(fmt.Sprintf("• К получению: %s\n\n", formatMoney(stats.UnpaidRevenue)))

	if stats.RepairRevenue > 0 || stats.PartsRevenue > 0 {
		result.WriteString("🔧 **По типам:**\n")
		result.WriteString(fmt.Sprintf("• За работу: %s\n", formatMoney(stats.RepairRevenue)))
		result.WriteString(fmt.Sprintf("• За запчасти: %s\n\n", formatMoney(stats.PartsRevenue)))
	}

	result.WriteString("📈 **Показатели:**\n")
	if stats.AverageOrderValue > 0 {
		result.WriteString(fmt.Sprintf("• Средний чек: %s\n", formatMoney(stats.AverageOrderValue)))
	}
	result.WriteString(fmt.Sprintf("• Процент оплат: %.1f%%\n", stats.PaymentRate))

	return result.String()
}

// FinancialStats содержит финансовую статистику
type FinancialStats struct {
	Period               string  `json:"period"`
	TotalOrdersWithPrice int     `json:"total_orders_with_price"`
	PaidOrders           int     `json:"paid_orders"`
	UnpaidOrders         int     `json:"unpaid_orders"`
	TotalRevenue         float64 `json:"total_revenue"`
	PaidRevenue          float64 `json:"paid_revenue"`
	UnpaidRevenue        float64 `json:"unpaid_revenue"`
	RepairRevenue        float64 `json:"repair_revenue"`
	PartsRevenue         float64 `json:"parts_revenue"`
	AverageOrderValue    float64 `json:"average_order_value"`
	PaymentRate          float64 `json:"payment_rate"`
}

// MasterFinancialStats содержит финансовую статистику по мастеру
type MasterFinancialStats struct {
	MasterID             uint    `json:"master_id"`
	MasterName           string  `json:"master_name"`
	TotalOrdersWithPrice int     `json:"total_orders_with_price"`
	PaidOrders           int     `json:"paid_orders"`
	TotalRevenue         float64 `json:"total_revenue"`
	PaidRevenue          float64 `json:"paid_revenue"`
	AverageOrderValue    float64 `json:"average_order_value"`
	PaymentRate          float64 `json:"payment_rate"`
}

// FormatMasterStatistics форматирует статистику мастеров
func (s *StatisticsService) FormatMasterStatistics(masterStats map[uint]*models.MasterStatistics, lang string) string {
	var result strings.Builder

	result.WriteString("👨‍🔧 **Статистика мастеров:**\n\n")

	for _, stats := range masterStats {
		if stats.TotalOrders == 0 {
			continue
		}

		result.WriteString(fmt.Sprintf("**%s**\n", stats.MasterName))
		result.WriteString(fmt.Sprintf("• Заказов: %d\n", stats.TotalOrders))
		result.WriteString(fmt.Sprintf("• Завершено: %d\n", stats.CompletedOrders))
		result.WriteString(fmt.Sprintf("• Выручка: %.2f сум\n", stats.TotalRevenue))

		if stats.AverageRepairTime > 0 {
			result.WriteString(fmt.Sprintf("• Ср. время ремонта: %.1f дней\n", stats.AverageRepairTime))
		}

		if stats.Rating > 0 {
			result.WriteString(fmt.Sprintf("• Рейтинг: %.1f/10\n", stats.Rating))
		}

		result.WriteString("\n")
	}

	return result.String()
}
