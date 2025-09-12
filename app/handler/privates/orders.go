package handlers

import (
	"fmt"
	"log"
	"math/rand"

	"idoctor-bot/app/api"
	"idoctor-bot/app/config"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
	"idoctor-bot/app/services"

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

	// Работаем с локальной базой данных
	var devices []models.Device
	var err error

	if user.Role == models.UserRoleAdmin {
		// Админ видит все заказы
		err = db.Preload("Customer").Preload("Master").Find(&devices).Error
	} else {
		// Мастер видит только свои заказы
		err = db.Preload("Customer").Preload("Master").Where("master_id = ?", user.ID).Find(&devices).Error
	}

	if err != nil {
		log.Printf("Error getting devices from database: %v", err)
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

	// Отправляем список заказов
	sendDevicesList(bot, update.Message.Chat.ID, devices, lang, user.Role == models.UserRoleAdmin)
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

// NewOrder создает новый заказ (доступно для админов и мастеров)
func NewOrder(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	// Получаем пользователя
	var user models.User
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.UserNotFound, lang))
		return
	}

	// Проверяем права доступа (админы и мастера)
	if user.Role != models.UserRoleAdmin && user.Role != models.UserRoleMaster {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ У вас нет доступа к созданию заказов",
			"uz": "❌ Sizda buyurtma yaratish huquqi yo'q",
			"en": "❌ You don't have permission to create orders",
		}, lang))
		return
	}

	// Начинаем процесс создания заказа
	stateService := services.NewStateService(db)
	orderData := &models.OrderData{}

	// Устанавливаем состояние ожидания имени клиента
	err := stateService.SetState(user.TelegramID, models.StateWaitingCustomerName, orderData)
	if err != nil {
		log.Printf("Error setting user state: %v", err)
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка системы",
			"uz": "❌ Tizim xatosi",
			"en": "❌ System error",
		}, lang))
		return
	}

	// Отправляем сообщение с просьбой ввести имя клиента
	messageText := i18n.GetText(map[string]string{
		"ru": "👤 Создание нового заказа\n\nШаг 1/5: Введите имя клиента\n\nПример: Иван Иванов",
		"uz": "👤 Yangi buyurtma yaratish\n\n1/5 qadam: Mijozning ismini kiriting\n\nMisol: Ivan Ivanov",
		"en": "👤 Creating new order\n\nStep 1/5: Enter customer name\n\nExample: John Doe",
	}, lang)

	// Кнопка отмены
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "❌ Отмена",
					"uz": "❌ Bekor qilish",
					"en": "❌ Cancel",
				}, lang),
				"cancel_new_order"),
		),
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending new order message: %v", err)
	}
}

// Masters управляет мастерами (только для админов)
func Masters(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	messageText := "👥 " + i18n.GetText(map[string]string{
		"ru": "Управление мастерами",
		"uz": "Ustalarni boshqarish",
		"en": "Masters management",
	}, lang) + "\n\n" + i18n.GetText(map[string]string{
		"ru": "Выберите действие для управления мастерами:",
		"uz": "Ustalarni boshqarish uchun amalni tanlang:",
		"en": "Choose an action for masters management:",
	}, lang)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
	msg.ReplyMarkup = getMastersMainKeyboard(lang)

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending masters menu: %v", err)
	}
}

// Analytics показывает главное меню аналитики (только для админов)
func Analytics(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	messageText := "📊 " + i18n.GetText(map[string]string{
		"ru": "Статистика и аналитика",
		"uz": "Statistika va analitika",
		"en": "Statistics and analytics",
	}, lang) + "\n\n" + i18n.GetText(map[string]string{
		"ru": "Выберите тип статистики или период для просмотра:",
		"uz": "Statistika turini yoki ko'rish uchun davrni tanlang:",
		"en": "Choose statistics type or period to view:",
	}, lang)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
	msg.ReplyMarkup = getStatisticsMainKeyboard(lang)

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending analytics menu: %v", err)
	}
}

// handleOrderDetails обрабатывает просмотр деталей заказа
func handleOrderDetails(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, user *models.User) {
	lang := user.GetLanguage()

	if !cfg.API.Enabled {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "API не подключен",
			"uz": "API ulanmagan",
			"en": "API not connected",
		}, lang))
		return
	}

	apiClient := api.NewAPIClient(cfg.API.BaseURL, cfg.API.APIKey)

	// Получаем детали заказа (пока мок - нужно добавить метод в API)
	devices, err := apiClient.GetDevices()
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения данных",
			"uz": "Ma'lumotlarni olishda xatolik",
			"en": "Error getting data",
		}, lang))
		return
	}

	// Ищем нужный заказ
	var device *api.Device
	for _, d := range devices {
		if d.ID == orderID {
			device = &d
			break
		}
	}

	if device == nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Заказ не найден",
			"uz": "Buyurtma topilmadi",
			"en": "Order not found",
		}, lang))
		return
	}

	// Формируем детальное описание
	messageText := fmt.Sprintf("📋 *%s #%d*\n\n",
		i18n.GetText(map[string]string{
			"ru": "Заказ",
			"uz": "Buyurtma",
			"en": "Order",
		}, lang), device.ID)

	messageText += fmt.Sprintf("📱 *%s %s*\n", device.Brand, device.Model)

	if device.SerialNumber != "" {
		messageText += fmt.Sprintf("🔢 %s: `%s`\n",
			i18n.GetText(map[string]string{
				"ru": "Серийный номер",
				"uz": "Seriya raqami",
				"en": "Serial number",
			}, lang), device.SerialNumber)
	}

	if device.Customer != nil {
		messageText += fmt.Sprintf("👤 *%s:* %s\n",
			i18n.GetText(map[string]string{
				"ru": "Клиент",
				"uz": "Mijoz",
				"en": "Customer",
			}, lang), device.Customer.Name)

		if device.Customer.PhoneNumber != "" {
			messageText += fmt.Sprintf("📞 %s\n", device.Customer.PhoneNumber)
		}
	}

	if device.Master != nil {
		messageText += fmt.Sprintf("👨‍🔧 *%s:* %s\n",
			i18n.GetText(map[string]string{
				"ru": "Мастер",
				"uz": "Usta",
				"en": "Master",
			}, lang), device.Master.Name)
	}

	messageText += fmt.Sprintf("⚡️ *%s:* %s\n",
		i18n.GetText(map[string]string{
			"ru": "Статус",
			"uz": "Status",
			"en": "Status",
		}, lang), getStatusText(device.Status, lang))

	if device.Price != nil {
		messageText += fmt.Sprintf("💰 *%s:* %.0f %s\n",
			i18n.GetText(map[string]string{
				"ru": "Цена",
				"uz": "Narx",
				"en": "Price",
			}, lang), *device.Price,
			i18n.GetText(map[string]string{
				"ru": "сум",
				"uz": "so'm",
				"en": "sum",
			}, lang))
	}

	if device.Issue != "" {
		messageText += fmt.Sprintf("\n🔍 *%s:*\n%s\n",
			i18n.GetText(map[string]string{
				"ru": "Описание проблемы",
				"uz": "Muammo tavsifi",
				"en": "Issue description",
			}, lang), device.Issue)
	}

	// Кнопки для управления заказом
	keyboard := getOrderManagementKeyboard(device, user, lang)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending order details: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// getOrderManagementKeyboard создает клавиатуру управления заказом
func getOrderManagementKeyboard(device *api.Device, user *models.User, lang string) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	// Кнопка изменения статуса (для мастеров и админов)
	if user.Role == models.UserRoleAdmin || (user.Role == models.UserRoleMaster && device.MasterID != nil && *device.MasterID == user.ID) {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🔄 Изменить статус",
					"uz": "🔄 Statusni o'zgartirish",
					"en": "🔄 Change status",
				}, lang),
				fmt.Sprintf("status_change_%d", device.ID)),
		})
	}

	// Кнопка установки цены (для админов)
	if user.Role == models.UserRoleAdmin {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "💰 Установить цену",
					"uz": "💰 Narx belgilash",
					"en": "💰 Set price",
				}, lang),
				fmt.Sprintf("price_set_%d", device.ID)),
		})
	}

	// Кнопка назначения мастера (для админов)
	if user.Role == models.UserRoleAdmin {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "👨‍🔧 Назначить мастера",
					"uz": "👨‍🔧 Usta tayinlash",
					"en": "👨‍🔧 Assign master",
				}, lang),
				fmt.Sprintf("master_assign_%d", device.ID)),
		})
	}

	// Кнопка возврата
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			i18n.GetText(map[string]string{
				"ru": "🔙 Назад",
				"uz": "🔙 Orqaga",
				"en": "🔙 Back",
			}, lang),
			"back_to_orders"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

func handleStatusChange(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, newStatus string, user *models.User) {
	lang := user.GetLanguage()

	if !cfg.API.Enabled {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "API не подключен",
			"uz": "API ulanmagan",
			"en": "API not connected",
		}, lang))
		return
	}

	apiClient := api.NewAPIClient(cfg.API.BaseURL, cfg.API.APIKey)

	// Обновляем статус через API
	err := apiClient.UpdateDeviceStatus(orderID, newStatus)
	if err != nil {
		log.Printf("Error updating device status: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка обновления статуса",
			"uz": "Statusni yangilashda xatolik",
			"en": "Error updating status",
		}, lang))
		return
	}

	// Отправляем подтверждение
	statusText := getStatusText(newStatus, lang)
	successMessage := fmt.Sprintf("%s\n\n✅ %s: %s",
		i18n.GetText(map[string]string{
			"ru": "Статус успешно обновлен!",
			"uz": "Status muvaffaqiyatli yangilandi!",
			"en": "Status updated successfully!",
		}, lang),
		i18n.GetText(map[string]string{
			"ru": "Новый статус",
			"uz": "Yangi status",
			"en": "New status",
		}, lang), statusText)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, successMessage)

	// Кнопка возврата к деталям заказа
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🔙 К деталям заказа",
					"uz": "🔙 Buyurtma tafsilotlariga",
					"en": "🔙 To order details",
				}, lang),
				fmt.Sprintf("order_details_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "📋 К списку заказов",
					"uz": "📋 Buyurtmalar ro'yxatiga",
					"en": "📋 To orders list",
				}, lang),
				"orders_refresh"),
		),
	)

	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending status update confirmation: %v", err)
	}

	// Логируем действие
	log.Printf("User %s (ID: %d) changed status of order %d to %s", user.Name, user.TelegramID, orderID, newStatus)

	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "✅ Статус обновлен",
		"uz": "✅ Status yangilandi",
		"en": "✅ Status updated",
	}, lang))
}

func handlePriceSet(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, user *models.User) {
	lang := user.GetLanguage()

	// Проверяем права доступа (только админы)
	if user.Role != models.UserRoleAdmin {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Нет доступа",
			"uz": "❌ Ruxsat yo'q",
			"en": "❌ Access denied",
		}, lang))
		return
	}

	if !cfg.API.Enabled {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "API не подключен",
			"uz": "API ulanmagan",
			"en": "API not connected",
		}, lang))
		return
	}

	// Устанавливаем состояние пользователя для ожидания ввода цены
	stateService := services.NewStateService(db)
	stateData := map[string]interface{}{
		"order_id": orderID,
		"action":   "price_set",
	}

	err := stateService.SetState(user.TelegramID, models.StateSettingPrice, stateData)
	if err != nil {
		log.Printf("Error setting user state: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка системы",
			"uz": "Tizim xatosi",
			"en": "System error",
		}, lang))
		return
	}

	// Отправляем сообщение с просьбой ввести цену
	messageText := fmt.Sprintf("%s #%d\n\n%s",
		i18n.GetText(map[string]string{
			"ru": "💰 Установка цены для заказа",
			"uz": "💰 Buyurtma uchun narx belgilash",
			"en": "💰 Setting price for order",
		}, lang), orderID,
		i18n.GetText(map[string]string{
			"ru": "Введите цену в сумах (только число):\n\nПример: 150000",
			"uz": "Narxni so'mda kiriting (faqat raqam):\n\nMisol: 150000",
			"en": "Enter price in sums (numbers only):\n\nExample: 150000",
		}, lang))

	// Кнопка отмены
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "❌ Отмена",
					"uz": "❌ Bekor qilish",
					"en": "❌ Cancel",
				}, lang),
				fmt.Sprintf("price_cancel_%d", orderID)),
		),
	)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending price request message: %v", err)
	}

	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "💰 Введите цену",
		"uz": "💰 Narxni kiriting",
		"en": "💰 Enter price",
	}, lang))
}

// showStatusChangeMenu показывает меню выбора статуса
func showStatusChangeMenu(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, user *models.User) {
	lang := user.GetLanguage()

	messageText := i18n.GetText(map[string]string{
		"ru": "🔄 Выберите новый статус:",
		"uz": "🔄 Yangi statusni tanlang:",
		"en": "🔄 Choose new status:",
	}, lang)

	// Создаем клавиатуру с вариантами статусов
	var keyboard [][]tgbotapi.InlineKeyboardButton

	statuses := []string{"received", "in_progress", "waiting_parts", "ready", "completed", "cancelled"}

	for _, status := range statuses {
		statusText := getStatusText(status, lang)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(statusText, fmt.Sprintf("status_set_%d_%s", orderID, status)),
		})
	}

	// Кнопка отмены
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			i18n.GetText(map[string]string{
				"ru": "❌ Отмена",
				"uz": "❌ Bekor qilish",
				"en": "❌ Cancel",
			}, lang),
			fmt.Sprintf("order_details_%d", orderID)),
	})

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending status menu: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleOrdersRefresh обновляет список заказов
func handleOrdersRefresh(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()

	if !cfg.API.Enabled {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "API не подключен",
			"uz": "API ulanmagan",
			"en": "API not connected",
		}, lang))
		return
	}

	apiClient := api.NewAPIClient(cfg.API.BaseURL, cfg.API.APIKey)

	var devices []api.Device
	var err error

	if user.Role == models.UserRoleAdmin {
		devices, err = apiClient.GetDevices()
	} else {
		devices, err = apiClient.GetDevicesByMaster(user.ID)
	}

	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения данных",
			"uz": "Ma'lumotlarni olishda xatolik",
			"en": "Error getting data",
		}, lang))
		return
	}

	// Отправляем обновленный список
	sendOrdersList(bot, callback.Message.Chat.ID, devices, lang, user.Role == models.UserRoleAdmin)

	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "✅ Обновлено",
		"uz": "✅ Yangilandi",
		"en": "✅ Updated",
	}, lang))
}

// handleHelpCallback обрабатывает callback кнопки справки
func handleHelpCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, action string, user *models.User) {
	lang := user.GetLanguage()

	switch action {
	case "contact":
		message := i18n.GetText(map[string]string{
			"ru": "📞 Контакты поддержки:\n\nДля связи с технической поддержкой обратитесь к администраторам системы.",
			"uz": "📞 Qo'llab-quvvatlash kontaktlari:\n\nTexnik yordam uchun tizim administratorlari bilan bog'laning.",
			"en": "📞 Support contacts:\n\nFor technical support, contact the system administrators.",
		}, lang)

		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, message)
		bot.Send(msg)

	case "about":
		message := i18n.GetText(map[string]string{
			"ru": "🔧 О системе:\n\niDoctor Bot v1.0\nСистема управления ремонтной мастерской с поддержкой мультиязычности.",
			"uz": "🔧 Tizim haqida:\n\niDoctor Bot v1.0\nKo'p tilli qo'llab-quvvatlash bilan ta'mirlash ustaxonasini boshqarish tizimi.",
			"en": "🔧 About the system:\n\niDoctor Bot v1.0\nRepair workshop management system with multi-language support.",
		}, lang)

		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, message)
		bot.Send(msg)
	}

	answerCallback(bot, callback.ID, "")
}

// handleMasterAssign обрабатывает назначение мастера
func handleMasterAssign(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, orderID uint, user *models.User) {
	// Временная заглушка
	answerCallback(bot, callback.ID, "Master assign - в разработке")
}

func handleBackToOrders(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	// Просто обновляем список заказов
	handleOrdersRefresh(bot, callback, cfg, db, user)
}

// getStatusText возвращает перевод статуса на указанный язык
// sendOrdersList отправляет список заказов с интерактивными кнопками
func sendOrdersList(bot *tgbotapi.BotAPI, chatID int64, devices []api.Device, lang string, isAdmin bool) {
	if len(devices) == 0 {
		message := i18n.GetText(map[string]string{
			"ru": "📋 Нет заказов для отображения",
			"uz": "📋 Ko'rsatish uchun buyurtmalar yo'q",
			"en": "📋 No orders to display",
		}, lang)
		msg := tgbotapi.NewMessage(chatID, message)
		bot.Send(msg)
		return
	}

	// Пагинация - показываем по 5 заказов
	pageSize := 5
	for page := 0; page*pageSize < len(devices); page++ {
		start := page * pageSize
		end := start + pageSize
		if end > len(devices) {
			end = len(devices)
		}

		pageDevices := devices[start:end]
		messageText := ""
		if page == 0 {
			messageText = i18n.GetText(map[string]string{
				"ru": "📋 Заказы:",
				"uz": "📋 Buyurtmalar:",
				"en": "📋 Orders:",
			}, lang) + "\n\n"
		}

		// Создаем inline клавиатуру
		var keyboard [][]tgbotapi.InlineKeyboardButton

		for i, device := range pageDevices {
			status := getStatusText(device.Status, lang)
			deviceText := fmt.Sprintf("%d. %s %s\n⚡️ %s\n",
				start+i+1, device.Brand, device.Model, status)

			if device.Price != nil {
				deviceText += fmt.Sprintf("💰 %.0f сум\n", *device.Price)
			}

			if device.Customer != nil {
				deviceText += fmt.Sprintf("👤 %s\n", device.Customer.Name)
			}

			messageText += deviceText + "\n"

			// Кнопка для детального просмотра
			buttonText := fmt.Sprintf("#%d - %s %s", device.ID, device.Brand, device.Model)
			if len(buttonText) > 60 {
				buttonText = buttonText[:57] + "..."
			}

			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("order_details_%d", device.ID)),
			})
		}

		// Добавляем кнопки управления если есть несколько страниц
		if len(devices) > pageSize {
			var paginationRow []tgbotapi.InlineKeyboardButton
			if page > 0 {
				paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("orders_page_%d", page-1)))
			}
			if end < len(devices) {
				paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("Далее ▶️", fmt.Sprintf("orders_page_%d", page+1)))
			}
			if len(paginationRow) > 0 {
				keyboard = append(keyboard, paginationRow)
			}
		}

		// Добавляем кнопку обновления
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🔄 Обновить",
					"uz": "🔄 Yangilash",
					"en": "🔄 Refresh",
				}, lang),
				"orders_refresh"),
		})

		msg := tgbotapi.NewMessage(chatID, messageText)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending orders page: %v", err)
		}

		// Если это первая страница из нескольких, делаем небольшую паузу
		if page == 0 && len(devices) > pageSize {
			break // Показываем только первую страницу, остальные по запросу
		}
	}
}

// sendDevicesList отправляет список устройств с интерактивными кнопками
func sendDevicesList(bot *tgbotapi.BotAPI, chatID int64, devices []models.Device, lang string, isAdmin bool) {
	if len(devices) == 0 {
		message := i18n.GetText(map[string]string{
			"ru": "📋 Нет заказов для отображения",
			"uz": "📋 Ko'rsatish uchun buyurtmalar yo'q",
			"en": "📋 No orders to display",
		}, lang)
		msg := tgbotapi.NewMessage(chatID, message)
		bot.Send(msg)
		return
	}

	// Пагинация - показываем по 5 заказов
	pageSize := 5
	for page := 0; page*pageSize < len(devices); page++ {
		start := page * pageSize
		end := start + pageSize
		if end > len(devices) {
			end = len(devices)
		}

		pageDevices := devices[start:end]
		messageText := ""
		if page == 0 {
			messageText = i18n.GetText(map[string]string{
				"ru": "📋 Заказы:",
				"uz": "📋 Buyurtmalar:",
				"en": "📋 Orders:",
			}, lang) + "\n\n"
		}

		// Создаем inline клавиатуру
		var keyboard [][]tgbotapi.InlineKeyboardButton

		for i, device := range pageDevices {
			status := getDeviceStatusText(device.Status, lang)
			deviceText := fmt.Sprintf("%d. %s %s\n⚡️ %s\n",
				start+i+1, device.Brand, device.Model, status)

			if device.RepairCost > 0 {
				deviceText += fmt.Sprintf("💰 %.0f сум\n", device.RepairCost)
			}

			if device.Customer != nil {
				deviceText += fmt.Sprintf("👤 %s\n", device.Customer.Name)
			}

			messageText += deviceText + "\n"

			// Кнопка для детального просмотра
			buttonText := fmt.Sprintf("#%d - %s %s", device.ID, device.Brand, device.Model)
			if len(buttonText) > 60 {
				buttonText = buttonText[:57] + "..."
			}

			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("device_details_%d", device.ID)),
			})
		}

		// Добавляем кнопки управления если есть несколько страниц
		if len(devices) > pageSize {
			var paginationRow []tgbotapi.InlineKeyboardButton
			if page > 0 {
				paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("devices_page_%d", page-1)))
			}
			if end < len(devices) {
				paginationRow = append(paginationRow, tgbotapi.NewInlineKeyboardButtonData("Далее ▶️", fmt.Sprintf("devices_page_%d", page+1)))
			}
			if len(paginationRow) > 0 {
				keyboard = append(keyboard, paginationRow)
			}
		}

		// Добавляем кнопку обновления
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🔄 Обновить",
					"uz": "🔄 Yangilash",
					"en": "🔄 Refresh",
				}, lang),
				"devices_refresh"),
		})

		msg := tgbotapi.NewMessage(chatID, messageText)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending devices page: %v", err)
		}

		// Если это первая страница из нескольких, делаем небольшую паузу
		if page == 0 && len(devices) > pageSize {
			break // Показываем только первую страницу, остальные по запросу
		}
	}
}

func getDeviceStatusText(status models.DeviceStatus, lang string) string {
	statusMap := map[models.DeviceStatus]map[string]string{
		models.DeviceStatusReceived: {
			"ru": "🆕 Принят",
			"uz": "🆕 Qabul qilindi",
			"en": "🆕 Received",
		},
		models.DeviceStatusInProgress: {
			"ru": "🔧 В работе",
			"uz": "🔧 Ishlanmoqda",
			"en": "🔧 In progress",
		},
		models.DeviceStatusWaitingParts: {
			"ru": "⏳ Ожидание запчастей",
			"uz": "⏳ Ehtiyot qismlar kutilmoqda",
			"en": "⏳ Waiting for parts",
		},
		models.DeviceStatusReady: {
			"ru": "✅ Готов",
			"uz": "✅ Tayyor",
			"en": "✅ Ready",
		},
		models.DeviceStatusCompleted: {
			"ru": "📦 Выдан",
			"uz": "📦 Berildi",
			"en": "📦 Completed",
		},
		models.DeviceStatusCancelled: {
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
	return status.String() // fallback
}

// generateOrderCode генерирует уникальный код заказа
func generateOrderCode(db *gorm.DB) (string, error) {
	var code string
	for i := 0; i < 10; i++ { // Максимум 10 попыток
		// Генерация кода: формат ID-XXXX (например, ID-1234)
		code = fmt.Sprintf("ID-%04d", rand.Intn(10000))

		// Проверяем уникальность
		var exists bool
		err := db.Model(&models.Device{}).Select("count(*) > 0").Where("code = ?", code).Find(&exists).Error
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("не удалось сгенерировать уникальный код")
}

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
