package handlers

import (
	"fmt"
	"log"

	"idoctor-bot/app/api"
	"idoctor-bot/app/config"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// MyOrders показывает заказы пользователя
func MyOrders(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	// Получаем пользователя
	var user models.User
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.UserNotFound, lang))
		return
	}

	if cfg.API.Enabled {
		// Используем API
		apiClient := api.NewAPIClient(cfg.API.BaseURL, cfg.API.APIKey)
		devices, err := apiClient.GetDevicesByMaster(user.ID)
		if err != nil {
			log.Printf("Error getting devices from API: %v", err)
			sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
				"ru": "❌ Ошибка получения данных",
				"uz": "❌ Ma'lumotlarni olishda xatolik",
				"en": "❌ Error getting data",
			}, lang))
			return
		}

		if len(devices) == 0 {
			sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
				"ru": "📋 У вас пока нет заказов",
				"uz": "📋 Sizda hozircha buyurtmalar yo'q",
				"en": "📋 You don't have any orders yet",
			}, lang))
			return
		}

		// Формируем список заказов
		messageText := i18n.GetText(map[string]string{
			"ru": "📋 Ваши заказы:",
			"uz": "📋 Sizning buyurtmalaringiz:",
			"en": "📋 Your orders:",
		}, lang) + "\n\n"

		for i, device := range devices {
			if i >= 10 { // Ограничиваем количество
				break
			}
			status := getStatusText(device.Status, lang)
			messageText += fmt.Sprintf("%d. %s %s\n⚡️ %s: %s\n",
				i+1, device.Brand, device.Model,
				i18n.GetText(map[string]string{"ru": "Статус", "uz": "Status", "en": "Status"}, lang),
				status)
			if device.Price != nil {
				messageText += fmt.Sprintf("💰 %s: %.2f\n",
					i18n.GetText(map[string]string{"ru": "Цена", "uz": "Narx", "en": "Price"}, lang),
					*device.Price)
			}
			messageText += "\n"
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending orders: %v", err)
		}
	} else {
		// Заглушка
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "📋 Ваши заказы (функция в разработке)",
			"uz": "Sizning buyurtmalaringiz (funksiya ishlab chiqilmoqda)",
			"en": "Your orders (function in development)",
		}, lang))
	}
}

// AllOrders показывает все заказы (только для админов)
func AllOrders(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	if cfg.API.Enabled {
		apiClient := api.NewAPIClient(cfg.API.BaseURL, cfg.API.APIKey)
		devices, err := apiClient.GetDevices()
		if err != nil {
			log.Printf("Error getting all devices from API: %v", err)
			sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
				"ru": "❌ Ошибка получения данных",
				"uz": "❌ Ma'lumotlarni olishda xatolik",
				"en": "❌ Error getting data",
			}, lang))
			return
		}

		if len(devices) == 0 {
			sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
				"ru": "📊 Нет заказов в системе",
				"uz": "📊 Tizimda buyurtmalar yo'q",
				"en": "📊 No orders in the system",
			}, lang))
			return
		}

		// Статистика по статусам
		statusCount := make(map[string]int)
		for _, device := range devices {
			statusCount[device.Status]++
		}

		messageText := i18n.GetText(map[string]string{
			"ru": "📊 Статистика заказов:",
			"uz": "📊 Buyurtmalar statistikasi:",
			"en": "📊 Orders statistics:",
		}, lang) + "\n\n"

		messageText += fmt.Sprintf("%s: %d\n",
			i18n.GetText(map[string]string{"ru": "Всего", "uz": "Jami", "en": "Total"}, lang),
			len(devices))

		for status, count := range statusCount {
			statusText := getStatusText(status, lang)
			messageText += fmt.Sprintf("%s: %d\n", statusText, count)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending all orders: %v", err)
		}
	} else {
		// Заглушка
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "📊 Все заказы (функция в разработке)",
			"uz": "Barcha buyurtmalar (funksiya ishlab chiqilmoqda)",
			"en": "All orders (function in development)",
		}, lang))
	}
}

// NewOrder создает новый заказ (только для админов)
func NewOrder(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	// Временная заглушка
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "➕ "+i18n.GetText(map[string]string{
		"ru": "Создание нового заказа (функция в разработке)",
		"uz": "Yangi buyurtma yaratish (funksiya ishlab chiqilmoqda)",
		"en": "Creating new order (function in development)",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending new order: %v", err)
	}
}

// Masters управляет мастерами (только для админов)
func Masters(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	// Временная заглушка
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "👥 "+i18n.GetText(map[string]string{
		"ru": "Управление мастерами (функция в разработке)",
		"uz": "Ustalarni boshqarish (funksiya ishlab chiqilmoqda)",
		"en": "Managing masters (function in development)",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending masters: %v", err)
	}
}

// Analytics показывает аналитику (только для админов)
func Analytics(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	// Временная заглушка
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "📈 "+i18n.GetText(map[string]string{
		"ru": "Аналитика (функция в разработке)",
		"uz": "Analitika (funksiya ishlab chiqilmoqda)",
		"en": "Analytics (function in development)",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending analytics: %v", err)
	}
}

// Заглушки для обработчиков callback
func handleOrderDetails(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, user interface{}) {
	// Временная заглушка
	answerCallback(bot, callback.ID, "Order details - в разработке")
}

func handleStatusChange(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, status interface{}, user interface{}) {
	// Временная заглушка
	answerCallback(bot, callback.ID, "Status change - в разработке")
}

func handlePriceSet(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, user interface{}) {
	// Временная заглушка
	answerCallback(bot, callback.ID, "Price set - в разработке")
}

func handleBackToOrders(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user interface{}) {
	// Временная заглушка
	answerCallback(bot, callback.ID, "Back to orders - в разработке")
}

// getStatusText возвращает перевод статуса на указанный язык
func getStatusText(status string, lang string) string {
	statusMap := map[string]map[string]string{
		"received": {
			"ru": "🆕 Принят",
			"uz": "🆕 Qabul qilindi",
			"en": "🆕 Received",
		},
		"in_progress": {
			"ru": "🔧 В работе",
			"uz": "🔧 Ishlanmoqda",
			"en": "🔧 In progress",
		},
		"waiting_parts": {
			"ru": "⏳ Ожидание запчастей",
			"uz": "⏳ Ehtiyot qismlar kutilmoqda",
			"en": "⏳ Waiting for parts",
		},
		"ready": {
			"ru": "✅ Готов",
			"uz": "✅ Tayyor",
			"en": "✅ Ready",
		},
		"completed": {
			"ru": "📦 Выдан",
			"uz": "📦 Berildi",
			"en": "📦 Completed",
		},
		"cancelled": {
			"ru": "❌ Отменен",
			"uz": "❌ Bekor qilindi",
			"en": "❌ Cancelled",
		},
	}

	if statusTexts, exists := statusMap[status]; exists {
		if text, exists := statusTexts[lang]; exists {
			return text
		}
		return statusTexts["ru"] // fallback
	}
	return status // fallback
}
