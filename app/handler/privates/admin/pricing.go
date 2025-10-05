package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
	"idoctor-bot/app/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// HandlePricingMenu обрабатывает меню управления ценами
func HandlePricingMenu(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	userID := uint(callbackQuery.From.ID)

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin && user.Role != models.UserRoleMaster {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для управления ценами")
		return
	}

	pricingService := services.NewPricingService(db)

	// Получаем устройства для установки цен
	devices, err := pricingService.GetDevicesForPricing(userID, user.Role)
	if err != nil {
		log.Printf("Error getting devices for pricing: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения заказов")
		return
	}

	if len(devices) == 0 {
		msg := tgbotapi.NewMessage(callbackQuery.From.ID, "📋 Нет заказов для установки цен")
		bot.Send(msg)
		return
	}

	// Создаем клавиатуру с заказами
	keyboard := tgbotapi.NewInlineKeyboardMarkup()

	for _, device := range devices {
		deviceText := fmt.Sprintf("🆔 %s - %s %s", device.Code, device.Brand, device.Model)
		if device.Customer != nil {
			deviceText += fmt.Sprintf(" (%s)", device.Customer.Name)
		}

		button := tgbotapi.NewInlineKeyboardButtonData(
			deviceText,
			fmt.Sprintf("pricing_device_%d", device.ID),
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	// Добавляем кнопку "Назад"
	backButton := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "main_menu")
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []tgbotapi.InlineKeyboardButton{backButton})

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, "💰 Выберите заказ для установки цены:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// HandleDevicePricing обрабатывает выбор устройства для установки цены
func HandleDevicePricing(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	// Извлекаем ID устройства из callback data
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 3 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID устройства")
		return
	}

	pricingService := services.NewPricingService(db)
	device, err := pricingService.GetDeviceWithPricing(uint(deviceID))
	if err != nil {
		log.Printf("Error getting device: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения заказа")
		return
	}

	// Форматируем информацию об устройстве
	text := pricingService.FormatPricingInfo(device)

	// Создаем клавиатуру для управления ценами
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Цена ремонта", fmt.Sprintf("set_repair_price_%d", device.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🔩 Цена запчастей", fmt.Sprintf("set_parts_price_%d", device.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Статус оплаты", fmt.Sprintf("toggle_paid_%d", device.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к списку", "pricing_menu"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

// HandleSetRepairPrice обрабатывает установку цены ремонта
func HandleSetRepairPrice(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 4 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID устройства")
		return
	}

	// Сохраняем ID устройства в состоянии пользователя
	stateService := services.NewStateService(db)
	err = stateService.SetState(callbackQuery.From.ID, "awaiting_repair_price", map[string]interface{}{
		"device_id": deviceID,
	})
	if err != nil {
		log.Printf("Error setting state: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка сохранения состояния")
		return
	}

	msg := tgbotapi.NewMessage(callbackQuery.From.ID,
		"💰 Введите цену ремонта в сумах:\n\n"+
			"Примеры: 50000, 75000.50, 100000\n"+
			"Для отмены отправьте /cancel")
	bot.Send(msg)
}

// HandleSetPartsPrice обрабатывает установку цены запчастей
func HandleSetPartsPrice(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 4 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID устройства")
		return
	}

	// Сохраняем ID устройства в состоянии пользователя
	stateService := services.NewStateService(db)
	err = stateService.SetState(callbackQuery.From.ID, "awaitingParts_price", map[string]interface{}{
		"device_id": deviceID,
	})
	if err != nil {
		log.Printf("Error setting state: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка сохранения состояния")
		return
	}

	msg := tgbotapi.NewMessage(callbackQuery.From.ID,
		"🔩 Введите цену запчастей в сумах:\n\n"+
			"Примеры: 25000, 30000.50, 45000\n"+
			"Если запчасти не требуются, введите 0\n"+
			"Для отмены отправьте /cancel")
	bot.Send(msg)
}

// HandleTogglePaid обрабатывает переключение статуса оплаты
func HandleTogglePaid(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 3 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID устройства")
		return
	}

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения пользователя")
		return
	}

	pricingService := services.NewPricingService(db)

	// Получаем текущий статус
	device, err := pricingService.GetDeviceWithPricing(uint(deviceID))
	if err != nil {
		log.Printf("Error getting device: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения заказа")
		return
	}

	// Переключаем статус оплаты
	newPaidStatus := !device.IsPaid
	err = pricingService.SetPaidStatus(uint(deviceID), newPaidStatus, uint(callbackQuery.From.ID), user.Role)
	if err != nil {
		log.Printf("Error setting paid status: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, fmt.Sprintf("Ошибка: %s", err.Error()))
		return
	}

	// Получаем обновленную информацию
	device, _ = pricingService.GetDeviceWithPricing(uint(deviceID))

	statusText := "❌ Не оплачено"
	if device.IsPaid {
		statusText = "✅ Оплачено"
	}

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, fmt.Sprintf("💳 Статус оплаты изменен: %s", statusText))
	bot.Send(msg)

	// Возвращаемся к экрану устройства
	HandleDevicePricing(bot, callbackQuery, db, langCache)
}

// HandlePriceInput обрабатывает ввод цены пользователем
func HandlePriceInput(bot *tgbotapi.BotAPI, message *tgbotapi.Message, db *gorm.DB, langCache *i18n.LanguageCache) {
	stateService := services.NewStateService(db)
	state, err := stateService.GetState(message.From.ID)
	if err != nil {
		log.Printf("Error getting state: %v", err)
		return
	}

	if state == nil {
		return
	}

	var user models.User
	if err := db.First(&user, "telegram_id = ?", message.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		SendErrorMessage(bot, message.From.ID, "Ошибка получения пользователя")
		return
	}

	pricingService := services.NewPricingService(db)

	// Парсим цену
	price, err := pricingService.ParsePriceFromText(message.Text)
	if err != nil {
		SendErrorMessage(bot, message.From.ID, err.Error())
		return
	}

	// Получаем ID устройства из состояния
	var stateData map[string]interface{}
	if err := json.Unmarshal([]byte(state.Data), &stateData); err != nil {
		log.Printf("Error unmarshaling state data: %v", err)
		SendErrorMessage(bot, message.From.ID, "Ошибка обработки состояния")
		stateService.ClearState(message.From.ID)
		return
	}

	deviceIDFloat, ok := stateData["device_id"].(float64)
	if !ok {
		SendErrorMessage(bot, message.From.ID, "Ошибка получения ID устройства")
		stateService.ClearState(message.From.ID)
		return
	}
	deviceID := uint(deviceIDFloat)

	var successMsg string

	// Устанавливаем цену в зависимости от состояния
	switch state.State {
	case "awaiting_repair_price":
		err = pricingService.SetRepairPrice(deviceID, price, uint(message.From.ID), user.Role)
		successMsg = fmt.Sprintf("✅ Цена ремонта установлена: %.2f сум", price)

	case "awaitingParts_price":
		err = pricingService.SetPartsPrice(deviceID, price, uint(message.From.ID), user.Role)
		successMsg = fmt.Sprintf("✅ Цена запчастей установлена: %.2f сум", price)

	default:
		SendErrorMessage(bot, message.From.ID, "Неизвестное состояние")
		stateService.ClearState(message.From.ID)
		return
	}

	if err != nil {
		log.Printf("Error setting price: %v", err)
		SendErrorMessage(bot, message.From.ID, fmt.Sprintf("Ошибка: %s", err.Error()))
		return
	}

	// Очищаем состояние
	stateService.ClearState(message.From.ID)

	// Отправляем сообщение об успехе
	msg := tgbotapi.NewMessage(message.From.ID, successMsg)
	bot.Send(msg)

	// Показываем обновленную информацию об устройстве
	device, _ := pricingService.GetDeviceWithPricing(deviceID)
	text := pricingService.FormatPricingInfo(device)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Цена ремонта", fmt.Sprintf("set_repair_price_%d", device.ID)),
			tgbotapi.NewInlineKeyboardButtonData("🔩 Цена запчастей", fmt.Sprintf("set_parts_price_%d", device.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Статус оплаты", fmt.Sprintf("toggle_paid_%d", device.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к списку", "pricing_menu"),
		),
	)

	updateMsg := tgbotapi.NewMessage(message.From.ID, text)
	updateMsg.ReplyMarkup = keyboard
	updateMsg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(updateMsg)
}

// HandleFinancialStats обрабатывает показ финансовой статистики
func HandleFinancialStats(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для просмотра финансовой статистики")
		return
	}

	// Создаем клавиатуру с периодами
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "financial_stats_today"),
			tgbotapi.NewInlineKeyboardButtonData("Вчера", "financial_stats_yesterday"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Эта неделя", "financial_stats_week"),
			tgbotapi.NewInlineKeyboardButtonData("Этот месяц", "financial_stats_month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Этот год", "financial_stats_year"),
			tgbotapi.NewInlineKeyboardButtonData("Все время", "financial_stats_all_time"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, "💰 Выберите период для финансовой статистики:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// HandleFinancialStatsPeriod обрабатывает показ статистики за выбранный период
func HandleFinancialStatsPeriod(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) < 3 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	var period models.StatisticsPeriod
	switch parts[2] {
	case "today":
		period = models.PeriodToday
	case "yesterday":
		period = models.PeriodYesterday
	case "week":
		period = models.PeriodWeek
	case "month":
		period = models.PeriodMonth
	case "year":
		period = models.PeriodYear
	case "all":
		period = models.PeriodAllTime
	default:
		SendErrorMessage(bot, callbackQuery.From.ID, "Неизвестный период")
		return
	}

	statsService := services.NewStatisticsService(db)
	stats, err := statsService.GetFinancialStatistics(period)
	if err != nil {
		log.Printf("Error getting financial stats: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения статистики")
		return
	}

	text := statsService.FormatFinancialStats(stats, "ru")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к периодам", "financial_stats"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

// SendErrorMessage отправляет сообщение об ошибке
func SendErrorMessage(bot *tgbotapi.BotAPI, chatID int64, errorText string) {
	msg := tgbotapi.NewMessage(chatID, "❌ "+errorText)
	bot.Send(msg)
}
