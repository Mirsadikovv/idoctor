package models

import "time"

// OrderStatistics содержит статистику заказов
type OrderStatistics struct {
	TotalOrders       int                        `json:"total_orders"`
	OrdersByStatus    map[DeviceStatus]int       `json:"orders_by_status"`
	OrdersByPeriod    map[string]int             `json:"orders_by_period"`
	TotalRevenue      float64                    `json:"total_revenue"`
	AverageRepairTime float64                    `json:"average_repair_time"` // в днях
	MasterStats       map[uint]*MasterStatistics `json:"master_stats"`
	TopBrands         map[string]int             `json:"top_brands"`
	ProblemCategories map[string]int             `json:"problem_categories"`
}

// MasterStatistics содержит статистику по конкретному мастеру
type MasterStatistics struct {
	MasterID          uint    `json:"master_id"`
	MasterName        string  `json:"master_name"`
	TotalOrders       int     `json:"total_orders"`
	CompletedOrders   int     `json:"completed_orders"`
	TotalRevenue      float64 `json:"total_revenue"`
	AverageRepairTime float64 `json:"average_repair_time"` // в днях
	Rating            float64 `json:"rating"`
}

// PeriodStatistics содержит статистику за определенный период
type PeriodStatistics struct {
	Period            string               `json:"period"`
	StartDate         time.Time            `json:"start_date"`
	EndDate           time.Time            `json:"end_date"`
	TotalOrders       int                  `json:"total_orders"`
	CompletedOrders   int                  `json:"completed_orders"`
	CancelledOrders   int                  `json:"cancelled_orders"`
	TotalRevenue      float64              `json:"total_revenue"`
	AverageOrderValue float64              `json:"average_order_value"`
	OrdersByStatus    map[DeviceStatus]int `json:"orders_by_status"`
	OrdersByMaster    map[uint]int         `json:"orders_by_master"`
	TopBrands         map[string]int       `json:"top_brands"`
	ProblemTypes      map[string]int       `json:"problem_types"`
}

// StatisticsPeriod определяет периоды для статистики
type StatisticsPeriod string

const (
	PeriodToday     StatisticsPeriod = "today"
	PeriodYesterday StatisticsPeriod = "yesterday"
	PeriodWeek      StatisticsPeriod = "week"
	PeriodMonth     StatisticsPeriod = "month"
	PeriodYear      StatisticsPeriod = "year"
	PeriodAllTime   StatisticsPeriod = "all_time"
)

// GetPeriodName возвращает локализованное название периода
func (p StatisticsPeriod) GetName(lang string) string {
	switch p {
	case PeriodToday:
		switch lang {
		case "ru":
			return "Сегодня"
		case "uz":
			return "Bugun"
		default:
			return "Today"
		}
	case PeriodYesterday:
		switch lang {
		case "ru":
			return "Вчера"
		case "uz":
			return "Kecha"
		default:
			return "Yesterday"
		}
	case PeriodWeek:
		switch lang {
		case "ru":
			return "Эта неделя"
		case "uz":
			return "Bu hafta"
		default:
			return "This week"
		}
	case PeriodMonth:
		switch lang {
		case "ru":
			return "Этот месяц"
		case "uz":
			return "Bu oy"
		default:
			return "This month"
		}
	case PeriodYear:
		switch lang {
		case "ru":
			return "Этот год"
		case "uz":
			return "Bu yil"
		default:
			return "This year"
		}
	case PeriodAllTime:
		switch lang {
		case "ru":
			return "За все время"
		case "uz":
			return "Barcha vaqt"
		default:
			return "All time"
		}
	default:
		return string(p)
	}
}

// GetPeriodDates возвращает даты начала и окончания периода
func (p StatisticsPeriod) GetPeriodDates() (time.Time, time.Time) {
	now := time.Now()

	switch p {
	case PeriodToday:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)
		return start, end

	case PeriodYesterday:
		yesterday := now.AddDate(0, 0, -1)
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)
		return start, end

	case PeriodWeek:
		// Начало недели (понедельник)
		weekday := int(now.Weekday())
		if weekday == 0 { // Воскресенье = 0, делаем его 7
			weekday = 7
		}
		start := now.AddDate(0, 0, -weekday+1)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 7).Add(-time.Nanosecond)
		return start, end

	case PeriodMonth:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return start, end

	case PeriodYear:
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(1, 0, 0).Add(-time.Nanosecond)
		return start, end

	case PeriodAllTime:
		start := time.Date(2020, 1, 1, 0, 0, 0, 0, now.Location()) // Начало времен для системы
		end := now
		return start, end

	default:
		return now, now
	}
}
