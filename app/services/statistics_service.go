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
		"Экран": {"экран", "дисплей", "треснул", "разбит", "черный экран", "тачскрин", "сенсор"},
		"Батарея": {"батарея", "аккумулятор", "заряд", "разряжается", "не заряжается", "быстро садится"},
		"Кнопки": {"кнопка", "кнопки", "громкость", "включения", "домой", "назад"},
		"Камера": {"камера", "фотоаппарат", "не фотографирует", "размытые фото"},
		"Звук": {"звук", "динамик", "микрофон", "тихий", "не слышно", "шумы"},
		"Связь": {"сеть", "связь", "сигнал", "интернет", "wifi", "bluetooth", "не ловит"},
		"Зарядка": {"зарядка", "разъем", "usb", "не заряжается", "зарядное"},
		"Программное": {"зависает", "тормозит", "не включается", "перезагружается", "глючит", "обновление"},
		"Корпус": {"корпус", "задняя крышка", "царапины", "вмятины"},
		"Вода": {"вода", "намок", "упал в воду", "жидкость"},
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