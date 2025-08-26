package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"idoctor-bot/app/config"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
	"idoctor-bot/app/services"
	"idoctor-bot/app/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func Register(bot *tgbotapi.BotAPI, db *gorm.DB) {
	commands := utils.GetBotCommands()
	_, err := bot.Request(tgbotapi.SetMyCommandsConfig{
		Commands: commands,
	})
	if err != nil {
		log.Println("Failed to register commands:", err)
	}
}

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	if update.Message == nil {
		return
	}

	// Получаем пользователя и язык
	var user models.User
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		// Если пользователя нет, только команда /start должна работать
		if update.Message.IsCommand() && update.Message.Command() == "start" {
			Start(bot, update, cfg, db, langCache)
			return
		}
		lang := langCache.Get(update.Message.From.ID)
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.UserNotFound, lang))
		return
	}

	lang := user.GetLanguage()
	langCache.Set(update.Message.From.ID, lang)

	// Проверка состояния пользователя для обработки ввода
	if !update.Message.IsCommand() {
		// Проверяем, ожидается ли ввод от пользователя
		handled := handleUserState(bot, update, cfg, db, &user)
		if handled {
			return
		}
	}

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			Start(bot, update, cfg, db, langCache)
		case "menu":
			Menu(bot, update, cfg, db, langCache)
		case "orders":
			MyOrders(bot, update, cfg, db, langCache)
		case "help":
			Help(bot, update, langCache)
		case "lang":
			ShowLanguageMenu(bot, update, cfg, db, langCache)
		case "search":
			HandleSearch(bot, update, cfg, db, &user)
		default:
			Help(bot, update, langCache)
		}
	} else if update.Message != nil {
		// Обработка смены языка
		if update.Message.Text == "🇷🇺 Русский" || update.Message.Text == "🇺🇿 O'zbek" || update.Message.Text == "🇬🇧 English" {
			ChangeLanguage(bot, update, cfg, db, langCache)
			return
		}

		// Обработка кнопок меню
		switch update.Message.Text {
		case i18n.GetButton("my_orders", lang):
			MyOrders(bot, update, cfg, db, langCache)
		case i18n.GetButton("all_orders", lang):
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				AllOrders(bot, update, cfg, db, langCache)
			} else {
				MyOrders(bot, update, cfg, db, langCache)
			}
		case i18n.GetButton("new_order", lang):
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				NewOrder(bot, update, cfg, db, langCache)
			} else {
				sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.NoAccess, lang))
			}
		case i18n.GetButton("masters", lang):
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				Masters(bot, update, cfg, db, langCache)
			} else {
				sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.NoAccess, lang))
			}
		case i18n.GetButton("analytics", lang):
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				Analytics(bot, update, cfg, db, langCache)
			} else {
				sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.NoAccess, lang))
			}
		case i18n.GetButton("search", lang):
			HandleSearch(bot, update, cfg, db, &user)
		case i18n.GetButton("menu", lang):
			Menu(bot, update, cfg, db, langCache)
		case i18n.GetButton("change_language", lang):
			ShowLanguageMenu(bot, update, cfg, db, langCache)
		default:
			// Попробуем обработать как поиск
			handleTextSearch(bot, update, cfg, db, &user)
		}
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	// Проверяем пользователя
	var user models.User
	if err := db.Where("telegram_id = ?", callback.From.ID).First(&user).Error; err != nil {
		lang := langCache.Get(callback.From.ID)
		answerCallback(bot, callback.ID, i18n.GetText(i18n.UserNotFound, lang))
		return
	}

	lang := user.GetLanguage()
	langCache.Set(callback.From.ID, lang)

	// Парсим данные callback
	data := callback.Data
	parts := strings.Split(data, "_")

	if len(parts) < 1 {
		answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidDataFormat, lang))
		return
	}

	action := parts[0]

	switch action {
	case "order":
		if len(parts) >= 3 && parts[1] == "details" {
			orderID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			handleOrderDetails(bot, callback, cfg, db, uint(orderID), &user)
		}
	case "status":
		if len(parts) >= 3 && parts[1] == "change" {
			orderID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			showStatusChangeMenu(bot, callback, cfg, db, uint(orderID), &user)
		} else if len(parts) >= 4 && parts[1] == "set" {
			// status_set_{orderID}_{newStatus}
			orderID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			newStatus := parts[3]
			handleStatusChange(bot, callback, cfg, db, uint(orderID), newStatus, &user)
		}
	case "price":
		if len(parts) >= 3 && parts[1] == "set" {
			orderID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			handlePriceSet(bot, callback, cfg, db, uint(orderID), &user)
		}
	case "master":
		if len(parts) >= 3 && parts[1] == "assign" {
			orderID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			handleMasterAssign(bot, callback, cfg, db, uint(orderID), &user)
		}
	case "back":
		if len(parts) >= 3 && parts[1] == "to" && parts[2] == "orders" {
			handleBackToOrders(bot, callback, cfg, db, &user)
		}
	case "orders":
		if len(parts) >= 2 && parts[1] == "refresh" {
			handleOrdersRefresh(bot, callback, cfg, db, &user)
		} else if len(parts) >= 2 && parts[1] == "page" {
			// TODO: Handle pagination
			answerCallback(bot, callback.ID, "Pagination - в разработке")
		}
	case "help":
		if len(parts) >= 2 {
			handleHelpCallback(bot, callback, cfg, db, parts[1], &user)
		}
	case "confirm":
		if len(parts) >= 3 && parts[1] == "new" && parts[2] == "order" {
			handleConfirmOrder(bot, callback, cfg, db, &user)
		}
	case "cancel":
		if len(parts) >= 3 && parts[1] == "new" && parts[2] == "order" {
			handleCancelOrder(bot, callback, cfg, db, &user)
		}
	case "order_nav":
		if len(parts) >= 2 {
			// Обработка навигации по шагам создания заказа
			handleOrderNavigation(bot, callback, cfg, db, parts[1], &user)
		}
	default:
		answerCallback(bot, callback.ID, i18n.GetText(i18n.UnknownAction, lang))
	}
}

func answerCallback(bot *tgbotapi.BotAPI, callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	if _, err := bot.Request(callback); err != nil {
		log.Printf("Error answering callback: %v", err)
	}
}

// handleUserState обрабатывает состояния пользователя (ввод данных)
func handleUserState(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, user *models.User) bool {
	stateService := services.NewStateService(db)
	state, err := stateService.GetState(user.TelegramID)
	if err != nil {
		log.Printf("Error getting user state: %v", err)
		return false
	}

	// Если пользователь в состоянии idle, не обрабатываем
	if state.State == models.StateIdle {
		return false
	}

	lang := user.GetLanguage()
	inputText := strings.TrimSpace(update.Message.Text)

	// Обработка состояний создания заказа
	switch state.State {
	case models.StateWaitingCustomerName:
		return handleCustomerNameInput(bot, update, db, stateService, user, inputText, lang)
	case models.StateWaitingCustomerPhone:
		return handleCustomerPhoneInput(bot, update, db, stateService, user, inputText, lang)
	case models.StateWaitingDeviceBrand:
		return handleDeviceBrandInput(bot, update, db, stateService, user, inputText, lang)
	case models.StateWaitingDeviceModel:
		return handleDeviceModelInput(bot, update, db, stateService, user, inputText, lang)
	case models.StateWaitingDeviceIssue:
		return handleDeviceIssueInput(bot, update, db, stateService, user, inputText, lang)
	case models.StateSettingPrice:
		return handlePriceInput(bot, update, db, stateService, user, inputText, lang)
	default:
		return false
	}
}

// Обработчики для каждого шага создания заказа

// handleCustomerNameInput обработка ввода имени клиента
func handleCustomerNameInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, stateService *services.StateService, user *models.User, inputText, lang string) bool {
	if len(inputText) < 2 || len(inputText) > 100 {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Имя должно быть от 2 до 100 символов. Повторите ввод.",
			"uz": "❌ Ism 2 dan 100 gacha belgi bo'lishi kerak. Qayta kiriting.",
			"en": "❌ Name must be 2 to 100 characters long. Please try again.",
		}, lang))
		return true
	}

	// Получаем текущие данные заказа
	var orderData models.OrderData
	stateService.GetStateData(user.TelegramID, &orderData)

	// Сохраняем имя клиента
	orderData.CustomerName = inputText

	// Переходим к следующему шагу - ввод телефона
	stateService.SetState(user.TelegramID, models.StateWaitingCustomerPhone, &orderData)

	messageText := i18n.GetText(map[string]string{
		"ru": fmt.Sprintf("👤 Клиент: %s\n\n📞 Шаг 2/5: Введите номер телефона\n\nПример: +998901234567", inputText),
		"uz": fmt.Sprintf("👤 Mijoz: %s\n\n📞 2/5 qadam: Telefon raqamini kiriting\n\nMisol: +998901234567", inputText),
		"en": fmt.Sprintf("👤 Customer: %s\n\n📞 Step 2/5: Enter phone number\n\nExample: +998901234567", inputText),
	}, lang)

	// Кнопки назад и отмены
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "⬅️ Назад",
					"uz": "⬅️ Orqaga",
					"en": "⬅️ Back",
				}, lang),
				"order_nav_back_to_name"),
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

	bot.Send(msg)
	return true
}

// handleCustomerPhoneInput обработка ввода телефона клиента
func handleCustomerPhoneInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, stateService *services.StateService, user *models.User, inputText, lang string) bool {
	// Простая валидация номера телефона
	if len(inputText) < 9 || len(inputText) > 20 {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Неверный формат номера телефона. Повторите ввод.",
			"uz": "❌ Telefon raqam formati noto'g'ri. Qayta kiriting.",
			"en": "❌ Invalid phone number format. Please try again.",
		}, lang))
		return true
	}

	// Получаем текущие данные заказа
	var orderData models.OrderData
	stateService.GetStateData(user.TelegramID, &orderData)

	// Сохраняем телефон
	orderData.CustomerPhone = inputText

	// Переходим к следующему шагу - ввод бренда
	stateService.SetState(user.TelegramID, models.StateWaitingDeviceBrand, &orderData)

	messageText := i18n.GetText(map[string]string{
		"ru": fmt.Sprintf("👤 %s\n📞 %s\n\n📱 Шаг 3/5: Введите бренд устройства\n\nПример: Apple, Samsung, Xiaomi", orderData.CustomerName, orderData.CustomerPhone),
		"uz": fmt.Sprintf("👤 %s\n📞 %s\n\n📱 3/5 qadam: Qurilma brendini kiriting\n\nMisol: Apple, Samsung, Xiaomi", orderData.CustomerName, orderData.CustomerPhone),
		"en": fmt.Sprintf("👤 %s\n📞 %s\n\n📱 Step 3/5: Enter device brand\n\nExample: Apple, Samsung, Xiaomi", orderData.CustomerName, orderData.CustomerPhone),
	}, lang)

	// Кнопки назад и отмены
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "⬅️ Назад",
					"uz": "⬅️ Orqaga",
					"en": "⬅️ Back",
				}, lang),
				"order_nav_back_to_phone"),
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

	bot.Send(msg)
	return true
}

// handleDeviceBrandInput обработка ввода бренда устройства
func handleDeviceBrandInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, stateService *services.StateService, user *models.User, inputText, lang string) bool {
	if len(inputText) < 2 || len(inputText) > 50 {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Бренд должен быть от 2 до 50 символов. Повторите ввод.",
			"uz": "❌ Brend 2 dan 50 gacha belgi bo'lishi kerak. Qayta kiriting.",
			"en": "❌ Brand must be 2 to 50 characters long. Please try again.",
		}, lang))
		return true
	}

	var orderData models.OrderData
	stateService.GetStateData(user.TelegramID, &orderData)

	orderData.DeviceBrand = inputText
	stateService.SetState(user.TelegramID, models.StateWaitingDeviceModel, &orderData)

	messageText := i18n.GetText(map[string]string{
		"ru": fmt.Sprintf("👤 %s\n📞 %s\n📱 %s\n\n🏷️ Шаг 4/5: Введите модель устройства\n\nПример: iPhone 12, Galaxy S21, Redmi Note 10", orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand),
		"uz": fmt.Sprintf("👤 %s\n📞 %s\n📱 %s\n\n🏷️ 4/5 qadam: Qurilma modelini kiriting\n\nMisol: iPhone 12, Galaxy S21, Redmi Note 10", orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand),
		"en": fmt.Sprintf("👤 %s\n📞 %s\n📱 %s\n\n🏷️ Step 4/5: Enter device model\n\nExample: iPhone 12, Galaxy S21, Redmi Note 10", orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand),
	}, lang)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "⬅️ Назад",
					"uz": "⬅️ Orqaga",
					"en": "⬅️ Back",
				}, lang),
				"order_nav_back_to_brand"),
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

	bot.Send(msg)
	return true
}

// handleDeviceModelInput обработка ввода модели устройства
func handleDeviceModelInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, stateService *services.StateService, user *models.User, inputText, lang string) bool {
	if len(inputText) < 2 || len(inputText) > 100 {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Модель должна быть от 2 до 100 символов. Повторите ввод.",
			"uz": "❌ Model 2 dan 100 gacha belgi bo'lishi kerak. Qayta kiriting.",
			"en": "❌ Model must be 2 to 100 characters long. Please try again.",
		}, lang))
		return true
	}

	var orderData models.OrderData
	stateService.GetStateData(user.TelegramID, &orderData)

	orderData.DeviceModel = inputText
	stateService.SetState(user.TelegramID, models.StateWaitingDeviceIssue, &orderData)

	messageText := i18n.GetText(map[string]string{
		"ru": fmt.Sprintf("👤 %s\n📞 %s\n📱 %s %s\n\n🔧 Шаг 5/5: Опишите проблему\n\nПример: Не включается, разбит экран, быстро разряжается", orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel),
		"uz": fmt.Sprintf("👤 %s\n📞 %s\n📱 %s %s\n\n🔧 5/5 qadam: Muammoni tasvirlab bering\n\nMisol: Yoqilmaydi, ekran singan, tez quvvatsizlanadi", orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel),
		"en": fmt.Sprintf("👤 %s\n📞 %s\n📱 %s %s\n\n🔧 Step 5/5: Describe the problem\n\nExample: Won't turn on, broken screen, battery drains fast", orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel),
	}, lang)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "⬅️ Назад",
					"uz": "⬅️ Orqaga",
					"en": "⬅️ Back",
				}, lang),
				"order_nav_back_to_model"),
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

	bot.Send(msg)
	return true
}

// handleDeviceIssueInput обработка ввода описания проблемы
func handleDeviceIssueInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, stateService *services.StateService, user *models.User, inputText, lang string) bool {
	if len(inputText) < 5 || len(inputText) > 500 {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Описание проблемы должно быть от 5 до 500 символов. Повторите ввод.",
			"uz": "❌ Muammo tavsifi 5 dan 500 gacha belgi bo'lishi kerak. Qayta kiriting.",
			"en": "❌ Problem description must be 5 to 500 characters long. Please try again.",
		}, lang))
		return true
	}

	var orderData models.OrderData
	stateService.GetStateData(user.TelegramID, &orderData)

	orderData.DeviceIssue = inputText
	stateService.SetState(user.TelegramID, models.StateConfirmingOrder, &orderData)

	// Показываем подтверждение заказа
	messageText := i18n.GetText(map[string]string{
		"ru": fmt.Sprintf("✅ Подтверждение заказа:\n\n👤 Клиент: %s\n📞 Телефон: %s\n📱 Устройство: %s %s\n🔧 Проблема: %s\n\nПодтвердите создание заказа:", 
			orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel, orderData.DeviceIssue),
		"uz": fmt.Sprintf("✅ Buyurtmani tasdiqlash:\n\n👤 Mijoz: %s\n📞 Telefon: %s\n📱 Qurilma: %s %s\n🔧 Muammo: %s\n\nBuyurtma yaratishni tasdiqlang:",
			orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel, orderData.DeviceIssue),
		"en": fmt.Sprintf("✅ Order confirmation:\n\n👤 Customer: %s\n📞 Phone: %s\n📱 Device: %s %s\n🔧 Issue: %s\n\nConfirm order creation:",
			orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel, orderData.DeviceIssue),
	}, lang)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "✅ Подтвердить",
					"uz": "✅ Tasdiqlash",
					"en": "✅ Confirm",
				}, lang),
				"confirm_new_order"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "⬅️ Назад",
					"uz": "⬅️ Orqaga",
					"en": "⬅️ Back",
				}, lang),
				"order_nav_back_to_issue"),
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

	bot.Send(msg)
	return true
}

// handlePriceInput обработка ввода цены (для существующих заказов)
func handlePriceInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, stateService *services.StateService, user *models.User, inputText, lang string) bool {
	// TODO: Реализовать обработку ввода цены
	return false
}

// Обработчики callback для создания заказа

// handleConfirmOrder обработка подтверждения создания заказа
func handleConfirmOrder(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	stateService := services.NewStateService(db)
	
	// Получаем данные заказа
	var orderData models.OrderData
	err := stateService.GetStateData(user.TelegramID, &orderData)
	if err != nil {
		log.Printf("Error getting order data: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка получения данных",
			"uz": "❌ Ma'lumotlarni olishda xatolik",
			"en": "❌ Error getting data",
		}, lang))
		return
	}
	
	// Сохраняем заказ в базу данных
	err = saveOrderToDB(db, &orderData, user)
	if err != nil {
		log.Printf("Error saving order: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка сохранения",
			"uz": "❌ Saqlashda xatolik",
			"en": "❌ Error saving",
		}, lang))
		return
	}
	
	// Очищаем состояние пользователя
	stateService.ClearState(user.TelegramID)
	
	// Отправляем сообщение об успешном создании заказа
	messageText := i18n.GetText(map[string]string{
		"ru": fmt.Sprintf("✅ Заказ создан!\n\n👤 Клиент: %s\n📞 Телефон: %s\n📱 Устройство: %s %s\n🔧 Проблема: %s\n\nСтатус: 🆕 Принят в работу\n\nКлиент будет уведомлен о принятом заказе.",
			orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel, orderData.DeviceIssue),
		"uz": fmt.Sprintf("✅ Buyurtma yaratildi!\n\n👤 Mijoz: %s\n📞 Telefon: %s\n📱 Qurilma: %s %s\n🔧 Muammo: %s\n\nStatus: 🆕 Qabul qilindi\n\nMijoz buyurtma qabul qilingani haqida xabardor qilinadi.",
			orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel, orderData.DeviceIssue),
		"en": fmt.Sprintf("✅ Order created!\n\n👤 Customer: %s\n📞 Phone: %s\n📱 Device: %s %s\n🔧 Issue: %s\n\nStatus: 🆕 Received\n\nCustomer will be notified about accepted order.",
			orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, orderData.DeviceModel, orderData.DeviceIssue),
	}, lang)
	
	// Кнопка для возврата в меню
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🏠 Главное меню",
					"uz": "🏠 Asosiy menyu",
					"en": "🏠 Main menu",
				}, lang),
				"main_menu"),
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "📊 Мои заказы",
					"uz": "📊 Mening buyurtmalarim",
					"en": "📊 My orders",
				}, lang),
				"my_orders"),
		),
	)
	
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ReplyMarkup = keyboard
	
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending confirmation message: %v", err)
	}
	
	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "✅ Заказ создан",
		"uz": "✅ Buyurtma yaratildi",
		"en": "✅ Order created",
	}, lang))
}

// handleCancelOrder обработка отмены создания заказа
func handleCancelOrder(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	stateService := services.NewStateService(db)
	
	// Очищаем состояние пользователя
	stateService.ClearState(user.TelegramID)
	
	messageText := i18n.GetText(map[string]string{
		"ru": "❌ Создание заказа отменено",
		"uz": "❌ Buyurtma yaratish bekor qilindi",
		"en": "❌ Order creation cancelled",
	}, lang)
	
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending cancel message: %v", err)
	}
	
	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "❌ Отменено",
		"uz": "❌ Bekor qilindi",
		"en": "❌ Cancelled",
	}, lang))
}

// handleOrderNavigation обработка навигации по шагам создания заказа
func handleOrderNavigation(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, action string, user *models.User) {
	lang := user.GetLanguage()
	stateService := services.NewStateService(db)
	
	// Получаем текущие данные
	var orderData models.OrderData
	stateService.GetStateData(user.TelegramID, &orderData)
	
	switch action {
	case "back_to_name":
		stateService.SetState(user.TelegramID, models.StateWaitingCustomerName, &orderData)
	case "back_to_phone":
		stateService.SetState(user.TelegramID, models.StateWaitingCustomerPhone, &orderData)
	case "back_to_brand":
		stateService.SetState(user.TelegramID, models.StateWaitingDeviceBrand, &orderData)
	case "back_to_model":
		stateService.SetState(user.TelegramID, models.StateWaitingDeviceModel, &orderData)
	case "back_to_issue":
		stateService.SetState(user.TelegramID, models.StateWaitingDeviceIssue, &orderData)
	default:
		answerCallback(bot, callback.ID, i18n.GetText(i18n.UnknownAction, lang))
		return
	}
	
	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "⬅️ Назад",
		"uz": "⬅️ Orqaga",
		"en": "⬅️ Back",
	}, lang))
}

// saveOrderToDB сохраняет заказ в базу данных
func saveOrderToDB(db *gorm.DB, orderData *models.OrderData, createdBy *models.User) error {
	log.Printf("Saving order: Customer=%s, Phone=%s, Device=%s %s, Issue=%s, CreatedBy=%s",
		orderData.CustomerName, orderData.CustomerPhone, orderData.DeviceBrand, 
		orderData.DeviceModel, orderData.DeviceIssue, createdBy.Name)
	
	// 1. Найти или создать клиента
	var customer models.Customer
	result := db.Where("phone = ?", orderData.CustomerPhone).First(&customer)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Создаем нового клиента
			customer = models.Customer{
				Name:  orderData.CustomerName,
				Phone: orderData.CustomerPhone,
			}
			if err := db.Create(&customer).Error; err != nil {
				log.Printf("Error creating customer: %v", err)
				return fmt.Errorf("ошибка создания клиента: %v", err)
			}
			log.Printf("Created new customer with ID: %d", customer.ID)
		} else {
			log.Printf("Error finding customer: %v", result.Error)
			return fmt.Errorf("ошибка поиска клиента: %v", result.Error)
		}
	} else {
		// Обновляем имя клиента, если оно изменилось
		if customer.Name != orderData.CustomerName {
			customer.Name = orderData.CustomerName
			if err := db.Save(&customer).Error; err != nil {
				log.Printf("Error updating customer name: %v", err)
			}
		}
		log.Printf("Found existing customer with ID: %d", customer.ID)
	}
	
	// 2. Генерируем уникальный код заказа
	orderCode, err := generateUniqueOrderCode(db)
	if err != nil {
		log.Printf("Error generating order code: %v", err)
		return fmt.Errorf("ошибка генерации кода заказа: %v", err)
	}
	
	// 3. Создаем устройство
	device := models.Device{
		Code:         orderCode,
		CustomerID:   customer.ID,
		DeviceType:   "phone", // Тип устройства - телефон
		Brand:        orderData.DeviceBrand,
		Model:        orderData.DeviceModel,
		Problem:      orderData.DeviceIssue,
		Status:       models.DeviceStatusReceived,
		ReceivedAt:   time.Now(),
		WarrantyDays: 30, // 30 дней гарантии по умолчанию
	}
	
	if err := db.Create(&device).Error; err != nil {
		log.Printf("Error creating device: %v", err)
		return fmt.Errorf("ошибка создания устройства: %v", err)
	}
	
	log.Printf("Successfully created device with code: %s (ID: %d)", device.Code, device.ID)
	
	// 4. Создаем уведомление для администраторов
	if err := notifyAdminsAboutNewOrder(db, &device, &customer); err != nil {
		log.Printf("Error notifying admins: %v", err)
		// Не возвращаем ошибку, так как заказ уже создан
	}
	
	return nil
}

// generateUniqueOrderCode генерирует уникальный код заказа
func generateUniqueOrderCode(db *gorm.DB) (string, error) {
	const maxAttempts = 10
	
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Генерируем код в формате R + год (2 цифры) + месяц (2 цифры) + день (2 цифры) + случайные 4 цифры
		now := time.Now()
		randomPart := rand.Intn(9999)
		code := fmt.Sprintf("R%02d%02d%02d%04d", 
			now.Year()%100, 
			now.Month(), 
			now.Day(), 
			randomPart)
		
		// Проверяем уникальность
		var count int64
		err := db.Model(&models.Device{}).Where("code = ?", code).Count(&count).Error
		if err != nil {
			return "", err
		}
		
		if count == 0 {
			return code, nil
		}
	}
	
	return "", fmt.Errorf("не удалось сгенерировать уникальный код за %d попыток", maxAttempts)
}

// notifyAdminsAboutNewOrder уведомляет всех администраторов о новом заказе
func notifyAdminsAboutNewOrder(db *gorm.DB, device *models.Device, customer *models.Customer) error {
	// Находим всех администраторов
	var admins []models.User
	err := db.Where("role = ?", models.UserRoleAdmin).Find(&admins).Error
	if err != nil {
		return fmt.Errorf("ошибка поиска администраторов: %v", err)
	}
	
	if len(admins) == 0 {
		log.Printf("No admins found to notify")
		return nil
	}
	
	// Создаем уведомления для каждого администратора
	for _, admin := range admins {
		notification := models.Notification{
			UserID:  &admin.ID,
			Type:    "new_order",
			Title:   "Новый заказ",
			Message: fmt.Sprintf("Получен новый заказ %s от клиента %s (%s). Устройство: %s %s.", 
				device.Code, customer.Name, customer.Phone, device.Brand, device.Model),
			IsRead:  false,
			Data: map[string]interface{}{
				"device_id":   device.ID,
				"customer_id": customer.ID,
				"order_code":  device.Code,
			},
		}
		
		if err := db.Create(&notification).Error; err != nil {
			log.Printf("Error creating notification for admin %d: %v", admin.ID, err)
			continue
		}
		
		log.Printf("Created notification for admin %s (ID: %d)", admin.Name, admin.ID)
	}
	
	return nil
}

// HandleSearch запускает режим поиска
func HandleSearch(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	
	message := i18n.GetText(map[string]string{
		"ru": "🔍 Поиск устройств\n\nВведите:\n• Номер телефона клиента\n• ID устройства\n• Серийный номер\n• Модель устройства\n\nПример: +998901234567 или iPhone 12",
		"uz": "🔍 Qurilmalarni qidirish\n\nKiriting:\n• Mijozning telefon raqami\n• Qurilma ID\n• Seriya raqami\n• Qurilma modeli\n\nMisol: +998901234567 yoki iPhone 12",
		"en": "🔍 Device search\n\nEnter:\n• Customer phone number\n• Device ID\n• Serial number\n• Device model\n\nExample: +998901234567 or iPhone 12",
	}, lang)
	
	sendMessage(bot, update.Message.Chat.ID, message)
}

// handleTextSearch обрабатывает поиск по тексту
func handleTextSearch(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	query := strings.TrimSpace(update.Message.Text)
	
	if len(query) < 3 {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Слишком короткий запрос. Минимум 3 символа.",
			"uz": "❌ So'rov juda qisqa. Kamida 3 ta belgi.",
			"en": "❌ Query too short. Minimum 3 characters.",
		}, lang))
		return
	}
	
	// Пока отправляем заглушку
	sendMessage(bot, update.Message.Chat.ID, fmt.Sprintf(
		i18n.GetText(map[string]string{
			"ru": "🔍 Ищу: %s\n\n⚠️ Поиск пока в разработке",
			"uz": "🔍 Qidiryapman: %s\n\n⚠️ Qidiruv hali ishlab chiqilmoqda",
			"en": "🔍 Searching: %s\n\n⚠️ Search is under development",
		}, lang), query))
}
