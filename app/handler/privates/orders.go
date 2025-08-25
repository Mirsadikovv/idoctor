package handlers

import (
	"fmt"
	"log"

	"idoctor-bot/app/config"
	"idoctor-bot/app/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func MyOrders(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	var user models.User
	
	// Получаем пользователя из базы данных
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		sendMessage(bot, update.Message.Chat.ID, "❌ Пользователь не найден. Используйте /start для регистрации.")
		return
	}

	var devices []models.Device
	query := db.Preload("Customer").Preload("Master")

	if user.Role == models.UserRoleMaster {
		// Мастер видит только свои заказы
		query = query.Where("master_id = ?", user.ID)
	}

	if err := query.Find(&devices).Error; err != nil {
		log.Printf("Ошибка получения заказов: %v", err)
		sendMessage(bot, update.Message.Chat.ID, "❌ Ошибка получения заказов.")
		return
	}

	if len(devices) == 0 {
		var message string
		if user.Role == models.UserRoleMaster {
			message = "📋 У вас пока нет назначенных заказов."
		} else {
			message = "📋 В системе пока нет заказов."
		}
		sendMessage(bot, update.Message.Chat.ID, message)
		return
	}

	message := fmt.Sprintf("📋 %s (%d):\n\n", getOrdersTitle(user.Role), len(devices))

	for i, device := range devices {
		if i >= 10 { // Ограничиваем до 10 заказов на страницу
			message += fmt.Sprintf("... и еще %d заказов\n", len(devices)-i)
			break
		}

		message += fmt.Sprintf("🔹 #%s - %s\n", device.Code, device.Status.Text())
		if device.Customer != nil {
			message += fmt.Sprintf("👤 %s (%s)\n", device.Customer.Name, device.Customer.Phone)
		}
		message += fmt.Sprintf("📱 %s %s\n", device.Brand, device.Model)
		
		if device.DeadlineAt != nil {
			if device.IsOverdue() {
				message += fmt.Sprintf("⏰ ❗ПРОСРОЧЕН: %s\n", device.DeadlineAt.Format("02.01.2006"))
			} else {
				message += fmt.Sprintf("⏰ До: %s\n", device.DeadlineAt.Format("02.01.2006"))
			}
		}
		message += "\n"
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки списка заказов: %v", err)
	}
}

func AllOrders(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	var user models.User
	
	// Проверяем, что пользователь администратор
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		sendMessage(bot, update.Message.Chat.ID, "❌ Пользователь не найден.")
		return
	}

	if user.Role != models.UserRoleAdmin {
		sendMessage(bot, update.Message.Chat.ID, "❌ У вас нет доступа к этой функции.")
		return
	}

	var devices []models.Device
	if err := db.Preload("Customer").Preload("Master").Find(&devices).Error; err != nil {
		log.Printf("Ошибка получения всех заказов: %v", err)
		sendMessage(bot, update.Message.Chat.ID, "❌ Ошибка получения заказов.")
		return
	}

	if len(devices) == 0 {
		sendMessage(bot, update.Message.Chat.ID, "📊 В системе пока нет заказов.")
		return
	}

	// Группируем по статусам
	statusGroups := make(map[models.DeviceStatus][]models.Device)
	for _, device := range devices {
		statusGroups[device.Status] = append(statusGroups[device.Status], device)
	}

	message := fmt.Sprintf("📊 Все заказы в системе (%d):\n\n", len(devices))

	// Показываем статистику по статусам
	for status, devicesInStatus := range statusGroups {
		message += fmt.Sprintf("%s: %d\n", status.Text(), len(devicesInStatus))
	}

	message += "\n📋 Последние 10 заказов:\n\n"

	// Показываем последние 10 заказов
	for i := len(devices) - 1; i >= 0 && i >= len(devices)-10; i-- {
		device := devices[i]
		message += fmt.Sprintf("🔹 #%s - %s\n", device.Code, device.Status.Text())
		
		if device.Customer != nil {
			message += fmt.Sprintf("👤 %s\n", device.Customer.Name)
		}
		
		if device.Master != nil {
			message += fmt.Sprintf("🔧 %s\n", device.Master.FullName())
		}
		
		message += "\n"
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки всех заказов: %v", err)
	}
}

func NewOrder(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	sendMessage(bot, update.Message.Chat.ID, "➕ Создание нового заказа временно недоступно.\n\nИспользуйте веб-интерфейс для создания заказов.")
}

func Masters(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	var masters []models.User
	if err := db.Where("role = ? AND is_active = ?", models.UserRoleMaster, true).Find(&masters).Error; err != nil {
		log.Printf("Ошибка получения мастеров: %v", err)
		sendMessage(bot, update.Message.Chat.ID, "❌ Ошибка получения списка мастеров.")
		return
	}

	if len(masters) == 0 {
		sendMessage(bot, update.Message.Chat.ID, "👥 В системе пока нет зарегистрированных мастеров.")
		return
	}

	message := fmt.Sprintf("👥 Мастера в системе (%d):\n\n", len(masters))

	for _, master := range masters {
		var deviceCount int64
		db.Model(&models.Device{}).Where("master_id = ?", master.ID).Count(&deviceCount)

		message += fmt.Sprintf("🔨 %s\n", master.FullName())
		if master.Username != nil && *master.Username != "" {
			message += fmt.Sprintf("@%s\n", *master.Username)
		}
		message += fmt.Sprintf("📋 Заказов: %d\n\n", deviceCount)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки списка мастеров: %v", err)
	}
}

func Analytics(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	var stats struct {
		TotalOrders     int64
		CompletedOrders int64
		InProgressOrders int64
		OverdueOrders   int64
		TotalRevenue    float64
	}

	// Общее количество заказов
	db.Model(&models.Device{}).Count(&stats.TotalOrders)

	// Завершенные заказы
	db.Model(&models.Device{}).Where("status = ?", models.DeviceStatusCompleted).Count(&stats.CompletedOrders)

	// В работе
	db.Model(&models.Device{}).Where("status IN ?", []models.DeviceStatus{
		models.DeviceStatusInProgress,
		models.DeviceStatusWaitingParts,
	}).Count(&stats.InProgressOrders)

	// Просроченные заказы
	db.Model(&models.Device{}).Where("deadline_at < NOW() AND status NOT IN ?", []models.DeviceStatus{
		models.DeviceStatusCompleted,
		models.DeviceStatusCancelled,
	}).Count(&stats.OverdueOrders)

	// Общий доход (только с завершенных заказов)
	db.Model(&models.Device{}).Where("status = ? AND is_paid = ?", models.DeviceStatusCompleted, true).
		Select("COALESCE(SUM(total_cost), 0)").Row().Scan(&stats.TotalRevenue)

	message := "📈 Аналитика мастерской:\n\n"
	message += fmt.Sprintf("📊 Всего заказов: %d\n", stats.TotalOrders)
	message += fmt.Sprintf("✅ Завершено: %d\n", stats.CompletedOrders)
	message += fmt.Sprintf("⚙️ В работе: %d\n", stats.InProgressOrders)
	message += fmt.Sprintf("⏰ Просрочено: %d\n", stats.OverdueOrders)
	message += fmt.Sprintf("💰 Общий доход: %.2f сум\n\n", stats.TotalRevenue)

	// Процент завершенных заказов
	if stats.TotalOrders > 0 {
		completionRate := float64(stats.CompletedOrders) / float64(stats.TotalOrders) * 100
		message += fmt.Sprintf("📈 Процент завершенных: %.1f%%\n", completionRate)
	}

	sendMessage(bot, update.Message.Chat.ID, message)
}

func getOrdersTitle(role models.UserRole) string {
	if role == models.UserRoleAdmin {
		return "Все заказы"
	}
	return "Мои заказы"
}