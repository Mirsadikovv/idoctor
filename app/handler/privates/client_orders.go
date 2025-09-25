package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
	"idoctor-bot/app/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// HandleCreateOrder обрабатывает начало создания заказа клиентом
func HandleCreateOrder(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, langCache *i18n.LanguageCache) {
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

	// Устанавливаем состояние ожидания типа устройства
	stateService := services.NewStateService(db)
	stateService.SetState(userID, models.StateClientWaitingDeviceType, nil)

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.GetText(i18n.ClientOrderStart, lang))
	bot.Send(msg)

	// Отвечаем на callback
	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

// HandleClientOrderMessage обрабатывает сообщения клиента при создании заказа
func HandleClientOrderMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, langCache *i18n.LanguageCache) {
	userID := update.Message.From.ID
	lang := langCache.Get(userID)
	text := strings.TrimSpace(update.Message.Text)

	if text == "" {
		return
	}

	stateService := services.NewStateService(db)
	userState, err := stateService.GetState(userID)
	if err != nil {
		return
	}

	var orderData models.ClientOrderData
	if userState.Data != "" {
		json.Unmarshal([]byte(userState.Data), &orderData)
	}

	switch userState.State {
	case models.StateClientWaitingDeviceType:
		orderData.DeviceType = text
		stateService.SetState(userID, models.StateClientWaitingDeviceBrand, orderData)
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.ClientOrderDeviceBrand, lang)))

	case models.StateClientWaitingDeviceBrand:
		orderData.DeviceBrand = text
		stateService.SetState(userID, models.StateClientWaitingDeviceModel, orderData)
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.ClientOrderDeviceModel, lang)))

	case models.StateClientWaitingDeviceModel:
		orderData.DeviceModel = text
		stateService.SetState(userID, models.StateClientWaitingProblem, orderData)
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.ClientOrderProblem, lang)))

	case models.StateClientWaitingProblem:
		orderData.Problem = text
		stateService.SetState(userID, models.StateClientWaitingContactInfo, orderData)
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.ClientOrderContact, lang)))

	case models.StateClientWaitingContactInfo:
		lines := strings.Split(text, "\n")
		if len(lines) >= 2 {
			orderData.ContactName = strings.TrimSpace(lines[0])
			orderData.ContactPhone = strings.TrimSpace(lines[1])

			stateService.SetState(userID, models.StateClientConfirmingOrder, orderData)

			confirmText := fmt.Sprintf(i18n.GetText(i18n.ClientOrderConfirm, lang),
				orderData.DeviceType, orderData.DeviceBrand, orderData.DeviceModel,
				orderData.Problem, orderData.ContactName, orderData.ContactPhone)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, confirmText)
			msg.ReplyMarkup = getOrderConfirmKeyboard(lang)
			bot.Send(msg)
		} else {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.ClientOrderContact, lang)))
		}
	}
}

// HandleOrderConfirm обрабатывает подтверждение заказа
func HandleOrderConfirm(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, langCache *i18n.LanguageCache) {
	userID := update.CallbackQuery.From.ID
	lang := langCache.Get(userID)
	action := update.CallbackQuery.Data

	stateService := services.NewStateService(db)
	userState, err := stateService.GetState(userID)
	if err != nil {
		return
	}

	if userState.State != models.StateClientConfirmingOrder {
		return
	}

	if action == "confirm_order_yes" {
		var orderData models.ClientOrderData
		json.Unmarshal([]byte(userState.Data), &orderData)

		// Создаем или находим клиента
		customer, err := createOrFindCustomer(db, orderData.ContactName, orderData.ContactPhone)
		if err != nil {
			log.Printf("Ошибка создания клиента: %v", err)
			bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка создания заказа"))
			return
		}

		// Получаем пользователя для ClientID
		var user models.User
		if err := db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
			log.Printf("Пользователь не найден: %v", err)
			return
		}

		// Создаем устройство
		device := models.Device{
			Code:         generateClientOrderCode(),
			CustomerID:   customer.ID,
			ClientID:     &user.ID,
			DeviceType:   orderData.DeviceType,
			Brand:        orderData.DeviceBrand,
			Model:        orderData.DeviceModel,
			Problem:      orderData.Problem,
			Status:       models.DeviceStatusReceived,
		}

		if err := db.Create(&device).Error; err != nil {
			log.Printf("Ошибка создания устройства: %v", err)
			bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка создания заказа"))
			return
		}

		// Очищаем состояние
		stateService.ClearState(userID)

		// Отправляем подтверждение
		confirmMsg := fmt.Sprintf(i18n.GetText(i18n.ClientOrderCreated, lang), device.Code)
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, confirmMsg))

		// Уведомляем мастеров о новом заказе
		notifyMastersNewOrder(bot, db, &device, langCache)

	} else if action == "confirm_order_no" {
		// Отменяем заказ
		stateService.ClearState(userID)
		bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.GetText(i18n.ClientOrderCancelled, lang)))
	}

	// Отвечаем на callback
	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

// getOrderConfirmKeyboard возвращает клавиатуру подтверждения заказа
func getOrderConfirmKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.GetText(i18n.ConfirmYes, lang), "confirm_order_yes"),
			tgbotapi.NewInlineKeyboardButtonData(i18n.GetText(i18n.ConfirmNo, lang), "confirm_order_no"),
		),
	)
}

// createOrFindCustomer создает или находит клиента
func createOrFindCustomer(db *gorm.DB, name, phone string) (*models.Customer, error) {
	var customer models.Customer

	// Сначала пытаемся найти по телефону
	if err := db.Where("phone = ?", phone).First(&customer).Error; err == nil {
		return &customer, nil
	}

	// Если не найден, создаем нового
	customer = models.Customer{
		Name:  name,
		Phone: phone,
	}

	if err := db.Create(&customer).Error; err != nil {
		return nil, err
	}

	return &customer, nil
}

// generateOrderCode генерирует код заказа
func generateClientOrderCode() string {
	// Простая генерация кода заказа
	return fmt.Sprintf("CL%d", time.Now().Unix()%100000)
}

// notifyMastersNewOrder уведомляет всех активных мастеров о новом заказе
func notifyMastersNewOrder(bot *tgbotapi.BotAPI, db *gorm.DB, device *models.Device, langCache *i18n.LanguageCache) {
	var masters []models.User
	if err := db.Where("role = ? AND is_active = ?", models.UserRoleMaster, true).Find(&masters).Error; err != nil {
		log.Printf("Ошибка получения мастеров: %v", err)
		return
	}

	for _, master := range masters {
		lang := langCache.Get(master.TelegramID)

		text := fmt.Sprintf("🆕 Новый заказ!\n\n📋 Код: %s\n📱 Устройство: %s %s %s\n❗ Проблема: %s",
			device.Code, device.DeviceType, device.Brand, device.Model, device.Problem)

		msg := tgbotapi.NewMessage(master.TelegramID, text)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetText(i18n.AcceptOrder, lang), fmt.Sprintf("accept_order_%d", device.ID)),
				tgbotapi.NewInlineKeyboardButtonData("ℹ️ Подробно", fmt.Sprintf("order_details_%d", device.ID)),
			),
		)

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки уведомления мастеру %s: %v", master.Name, err)
		}
	}
}