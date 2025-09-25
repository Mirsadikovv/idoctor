package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// HandlePendingOrders показывает заказы в ожидании для мастеров
func HandlePendingOrders(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, langCache *i18n.LanguageCache) {
	userID := update.CallbackQuery.From.ID
	lang := langCache.Get(userID)

	// Проверяем роль пользователя
	var user models.User
	if err := db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
		log.Printf("Пользователь не найден: %v", err)
		return
	}

	if !user.IsMaster() {
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.GetText(i18n.NoAccess, lang)))
		return
	}

	// Получаем заказы без мастера со статусом "received"
	var devices []models.Device
	if err := db.Preload("Customer").Where("master_id IS NULL AND status = ?", models.DeviceStatusReceived).Find(&devices).Error; err != nil {
		log.Printf("Ошибка получения заказов: %v", err)
		return
	}

	if len(devices) == 0 {
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.GetText(i18n.NoAvailableOrders, lang))
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
			),
		)
		bot.Send(msg)
		return
	}

	text := fmt.Sprintf("⏳ %s:\n\n", i18n.GetText(i18n.PendingOrders, lang))
	for i, device := range devices {
		if i >= 5 { // Показываем максимум 5 заказов
			text += fmt.Sprintf("... и еще %d заказов\n", len(devices)-5)
			break
		}
		text += fmt.Sprintf("📋 %s - %s %s %s\n❗ %s\n\n", device.Code, device.DeviceType, device.Brand, device.Model, device.Problem)
	}

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = getPendingOrdersKeyboard(devices, lang)
	bot.Send(msg)

	// Отвечаем на callback
	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

// HandleAcceptOrder обрабатывает принятие заказа мастером
func HandleAcceptOrder(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, langCache *i18n.LanguageCache) {
	userID := update.CallbackQuery.From.ID
	lang := langCache.Get(userID)

	// Получаем ID заказа из callback data
	parts := strings.Split(update.CallbackQuery.Data, "_")
	if len(parts) != 3 {
		return
	}

	deviceID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return
	}

	// Проверяем роль пользователя
	var user models.User
	if err := db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
		log.Printf("Пользователь не найден: %v", err)
		return
	}

	if !user.IsMaster() || !user.IsActive {
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.GetText(i18n.NoAccess, lang)))
		return
	}

	// Получаем заказ
	var device models.Device
	if err := db.Preload("Customer").Preload("Client").Where("id = ?", deviceID).First(&device).Error; err != nil {
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Заказ не найден"))
		return
	}

	// Проверяем, что заказ еще не назначен
	if device.MasterID != nil {
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Заказ уже принят другим мастером"))
		return
	}

	// Назначаем мастера на заказ
	device.MasterID = &user.ID
	device.Status = models.DeviceStatusInProgress
	if err := db.Save(&device).Error; err != nil {
		log.Printf("Ошибка назначения мастера: %v", err)
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка принятия заказа"))
		return
	}

	// Отправляем подтверждение мастеру
	confirmText := fmt.Sprintf("✅ Вы приняли заказ %s\n\n📱 Устройство: %s %s %s\n❗ Проблема: %s\n\nСтатус изменен на 'В работе'",
		device.Code, device.DeviceType, device.Brand, device.Model, device.Problem)

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, confirmText)
	bot.Send(msg)

	// Уведомляем клиента о принятии заказа
	if device.Client != nil {
		clientLang := langCache.Get(device.Client.TelegramID)
		notifyText := fmt.Sprintf(i18n.GetText(i18n.OrderAccepted, clientLang)+"\n\n📋 Заказ: %s\n👨‍🔧 Мастер: %s",
			device.Code, user.FullName())

		clientMsg := tgbotapi.NewMessage(device.Client.TelegramID, notifyText)
		if _, err := bot.Send(clientMsg); err != nil {
			log.Printf("Ошибка отправки уведомления клиенту: %v", err)
		}
	}

	// Уведомляем других мастеров, что заказ уже принят
	notifyOtherMastersOrderTaken(bot, db, &device, user.ID, langCache)

	// Отвечаем на callback
	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Заказ принят"))
}

// HandleMyOrders показывает заказы мастера
func HandleMyOrders(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, langCache *i18n.LanguageCache) {
	userID := update.CallbackQuery.From.ID
	lang := langCache.Get(userID)

	// Проверяем роль пользователя
	var user models.User
	if err := db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
		log.Printf("Пользователь не найден: %v", err)
		return
	}

	if !user.IsMaster() {
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.GetText(i18n.NoAccess, lang)))
		return
	}

	// Получаем заказы мастера
	var devices []models.Device
	if err := db.Preload("Customer").Where("master_id = ?", user.ID).Order("created_at DESC").Limit(10).Find(&devices).Error; err != nil {
		log.Printf("Ошибка получения заказов мастера: %v", err)
		return
	}

	if len(devices) == 0 {
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "У вас пока нет заказов"))
		return
	}

	text := fmt.Sprintf("📋 %s:\n\n", i18n.GetText(i18n.MyOrders, lang))
	for _, device := range devices {
		statusIcon := getStatusIcon(device.Status)
		text += fmt.Sprintf("%s %s - %s %s\n💰 %s\n\n", statusIcon, device.Code, device.DeviceType, device.Brand, formatMoney(device.TotalCost))
	}

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "my_orders"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)
	bot.Send(msg)

	// Отвечаем на callback
	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

// HandleClientOrders показывает заказы клиента
func HandleClientOrders(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, langCache *i18n.LanguageCache) {
	userID := update.CallbackQuery.From.ID
	lang := langCache.Get(userID)

	// Проверяем роль пользователя
	var user models.User
	if err := db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
		log.Printf("Пользователь не найден: %v", err)
		return
	}

	if !user.IsClient() {
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.GetText(i18n.NoAccess, lang)))
		return
	}

	// Получаем заказы клиента
	var devices []models.Device
	if err := db.Preload("Customer").Preload("Master").Where("client_id = ?", user.ID).Order("created_at DESC").Find(&devices).Error; err != nil {
		log.Printf("Ошибка получения заказов клиента: %v", err)
		return
	}

	if len(devices) == 0 {
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "У вас пока нет заказов")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetText(i18n.CreateOrder, lang), "create_order"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
			),
		)
		bot.Send(msg)
		return
	}

	text := fmt.Sprintf("📋 %s:\n\n", i18n.GetText(i18n.MyOrders, lang))
	for _, device := range devices {
		statusIcon := getStatusIcon(device.Status)
		masterName := "Не назначен"
		if device.Master != nil {
			masterName = device.Master.FullName()
		}

		text += fmt.Sprintf("%s %s - %s %s\n👨‍🔧 Мастер: %s\n💰 %s\n\n",
			statusIcon, device.Code, device.DeviceType, device.Brand, masterName, formatMoney(device.TotalCost))
	}

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "client_orders"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.GetText(i18n.CreateOrder, lang), "create_order"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "main_menu"),
		),
	)
	bot.Send(msg)

	// Отвечаем на callback
	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

// notifyOtherMastersOrderTaken уведомляет других мастеров о том, что заказ принят
func notifyOtherMastersOrderTaken(bot *tgbotapi.BotAPI, db *gorm.DB, device *models.Device, takenByMasterID uint, langCache *i18n.LanguageCache) {
	var masters []models.User
	if err := db.Where("role = ? AND is_active = ? AND id != ?", models.UserRoleMaster, true, takenByMasterID).Find(&masters).Error; err != nil {
		return
	}

	for _, master := range masters {
		text := fmt.Sprintf("ℹ️ Заказ %s уже принят другим мастером", device.Code)

		if _, err := bot.Send(tgbotapi.NewMessage(master.TelegramID, text)); err != nil {
			log.Printf("Ошибка отправки уведомления мастеру: %v", err)
		}
	}
}

// getStatusIcon возвращает иконку для статуса заказа
func getStatusIcon(status models.DeviceStatus) string {
	switch status {
	case models.DeviceStatusReceived:
		return "🆕"
	case models.DeviceStatusInProgress:
		return "🔧"
	case models.DeviceStatusWaitingParts:
		return "⏳"
	case models.DeviceStatusReady:
		return "✅"
	case models.DeviceStatusCompleted:
		return "📦"
	case models.DeviceStatusCancelled:
		return "❌"
	default:
		return "❓"
	}
}

// formatMoney форматирует сумму денег
func formatMoney(amount float64) string {
	if amount == 0 {
		return "Не указано"
	}
	return fmt.Sprintf("%.2f сум", amount)
}