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
		
		// Проверяем, если пользователь вводит цену
		HandlePriceInput(bot, update.Message, db, langCache)
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
			if user.Role == models.UserRoleAdmin || user.Role == models.UserRoleMaster {
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

	// Специальная обработка для составных callback data без разделения по параметрам
	simpleCallbacks := []string{
		"main_menu", "back_to_masters", "back_to_orders", "back_to_stats",
		"all_orders", "new_order", "main_masters", "analytics", "search", "my_orders",
		"stats_general", "stats_masters", "stats_refresh", "masters_refresh",
		"masters_add", "help_contact", "orders_refresh", "masters_list_all", "masters_list_active",
		"stats_period_today", "stats_period_yesterday", "stats_period_week",
		"stats_period_month", "stats_period_year", "stats_period_all_time", "devices_refresh",
		"pricing_menu", "financial_stats",
	}

	for _, callback := range simpleCallbacks {
		if data == callback {
			action = data
			break
		}
	}

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
	case "stats":
		handleStatisticsCallback(bot, callback, cfg, db, parts, &user)
	case "back_to_stats":
		handleBackToStatistics(bot, callback, cfg, db, &user)
	case "masters":
		handleMastersCallback(bot, callback, cfg, db, parts, &user)
	case "master_action":
		handleMasterCallback(bot, callback, cfg, db, parts, &user)
	case "assign":
		handleAssignCallback(bot, callback, cfg, db, parts, &user)
	case "back_to_masters":
		handleBackToMasters(bot, callback, cfg, db, &user)
	case "main_menu":
		handleMainMenuCallback(bot, callback, cfg, db, &user)
	case "all_orders":
		if utils.IsAdmin(callback.From.ID, cfg) {
			handleAllOrdersCallback(bot, callback, cfg, db, &user)
		} else {
			answerCallback(bot, callback.ID, i18n.GetText(i18n.NoAccess, lang))
		}
	case "new_order":
		if utils.IsAdmin(callback.From.ID, cfg) {
			handleNewOrderCallback(bot, callback, cfg, db, &user)
		} else {
			answerCallback(bot, callback.ID, i18n.GetText(i18n.NoAccess, lang))
		}
	case "my_orders":
		handleMyOrdersCallback(bot, callback, cfg, db, &user)
	case "main_masters":
		if utils.IsAdmin(callback.From.ID, cfg) {
			handleMastersMenuCallback(bot, callback, cfg, db, &user)
		} else {
			answerCallback(bot, callback.ID, i18n.GetText(i18n.NoAccess, lang))
		}
	case "analytics":
		if utils.IsAdmin(callback.From.ID, cfg) {
			handleAnalyticsCallback(bot, callback, cfg, db, &user)
		} else {
			answerCallback(bot, callback.ID, i18n.GetText(i18n.NoAccess, lang))
		}
	case "pricing_menu":
		HandlePricingMenu(bot, callback, db, langCache)
	case "financial_stats":
		HandleFinancialStats(bot, callback, db, langCache)
	case "device":
		if len(parts) >= 3 && parts[1] == "details" {
			deviceID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			handleDeviceDetails(bot, callback, cfg, db, uint(deviceID), &user)
		} else if len(parts) >= 4 && parts[1] == "status" && parts[2] == "change" {
			deviceID, err := strconv.ParseUint(parts[3], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			handleDeviceStatusChangeMenu(bot, callback, cfg, db, uint(deviceID), &user)
		} else if len(parts) >= 5 && parts[1] == "status" && parts[2] == "set" {
			deviceID, err := strconv.ParseUint(parts[3], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			newStatus := parts[4]
			handleDeviceStatusSet(bot, callback, cfg, db, uint(deviceID), newStatus, &user)
		}
	case "search":
		handleSearchCallback(bot, callback, cfg, db, &user)
	case "back_to_orders":
		handleBackToOrdersCallback(bot, callback, cfg, db, &user)
	case "help_contact":
		handleHelpContactCallback(bot, callback, &user)
	case "orders_refresh":
		handleOrdersRefreshCallback(bot, callback, cfg, db, &user)
	case "devices_refresh":
		handleMyOrdersCallback(bot, callback, cfg, db, &user)
	default:
		// Проверяем составные callback data
		if strings.HasPrefix(action, "change_status_") {
			handleChangeStatusCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "set_price_") {
			handleSetPriceCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "assign_master_") {
			handleAssignMasterCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "edit_order_") {
			handleEditOrderCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "details_") {
			handleOrderDetailsCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "order_details_") {
			handleOrderDetailsCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "orders_page_") {
			handleOrdersPageCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "status_set_") {
			handleStatusSetCallbackComposite(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(action, "order_nav_") {
			// Обрабатываем order_nav_back_to_* callback data
			parts := strings.Split(data, "_")
			if len(parts) >= 4 && parts[2] == "back" && parts[3] == "to" {
				handleOrderNavigation(bot, callback, cfg, db, "back_to_"+parts[4], &user)
			}
		} else if action == "stats_general" {
			handleStatsGeneralCallback(bot, callback, cfg, db, &user)
		} else if action == "stats_masters" {
			handleStatsMastersCallback(bot, callback, cfg, db, &user)
		} else if strings.HasPrefix(action, "stats_period_") {
			handleStatsPeriodCallback(bot, callback, cfg, db, &user, action)
		} else if action == "stats_refresh" {
			handleAnalyticsCallback(bot, callback, cfg, db, &user)
		} else if action == "masters_refresh" {
			handleMastersMenuCallback(bot, callback, cfg, db, &user)
		} else if action == "masters_list_all" {
			handleMastersListAllCallback(bot, callback, cfg, db, &user)
		} else if action == "masters_list_active" {
			handleMastersListActiveCallback(bot, callback, cfg, db, &user)
		} else if action == "masters_add" {
			handleMastersAddCallback(bot, callback, cfg, db, &user)
		} else if strings.HasPrefix(action, "master_action_") {
			// Обрабатываем master_action_* callback data
			handleMasterActionCallback(bot, callback, cfg, db, &user, action)
		} else if strings.HasPrefix(data, "device_status_set_") {
			// Обработка device_status_set_<deviceID>_<status>
			log.Printf("Processing device_status_set callback: data='%s'", data)
			parts := strings.Split(data, "_")
			log.Printf("Parsed parts: %v, len=%d", parts, len(parts))
			if len(parts) >= 5 {
				deviceID, err := strconv.ParseUint(parts[3], 10, 32)
				if err != nil {
					answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
					return
				}
				newStatus := parts[4]
				log.Printf("Calling handleDeviceStatusSet with deviceID=%d, status='%s'", uint(deviceID), newStatus)
				handleDeviceStatusSet(bot, callback, cfg, db, uint(deviceID), newStatus, &user)
			} else {
				log.Printf("Invalid device_status_set format: expected >=5 parts, got %d", len(parts))
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidDataFormat, lang))
			}
		} else if action == "pricing_menu" {
			HandlePricingMenu(bot, callback, db, langCache)
		} else if action == "financial_stats" {
			HandleFinancialStats(bot, callback, db, langCache)
		} else if strings.HasPrefix(action, "pricing_device_") {
			HandleDevicePricing(bot, callback, db, langCache)
		} else if strings.HasPrefix(action, "set_repair_price_") {
			HandleSetRepairPrice(bot, callback, db, langCache)
		} else if strings.HasPrefix(action, "set_parts_price_") {
			HandleSetPartsPrice(bot, callback, db, langCache)
		} else if strings.HasPrefix(action, "toggle_paid_") {
			HandleTogglePaid(bot, callback, db, langCache)
		} else {
			answerCallback(bot, callback.ID, i18n.GetText(i18n.UnknownAction, lang))
		}
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
	case models.StateWaitingMasterTelegramID:
		return handleMasterTelegramIDInput(bot, update, db, stateService, user, inputText, lang)
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
	orderCode, err := generateOrderCode(db)
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

	// Если создается мастером, автоматически назначаем его ответственным
	if createdBy.Role == models.UserRoleMaster {
		device.MasterID = &createdBy.ID
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
			UserID: &admin.ID,
			Type:   "new_order",
			Title:  "Новый заказ",
			Message: fmt.Sprintf("Получен новый заказ %s от клиента %s (%s). Устройство: %s %s.",
				device.Code, customer.Name, customer.Phone, device.Brand, device.Model),
			IsRead: false,
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

// handleStatisticsCallback обрабатывает callback'и статистики
func handleStatisticsCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, parts []string, user *models.User) {
	lang := user.GetLanguage()

	if len(parts) < 2 {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный формат данных",
			"uz": "Noto'g'ri ma'lumot formati",
			"en": "Invalid data format",
		}, lang))
		return
	}

	action := parts[1]

	switch action {
	case "general":
		showGeneralStatistics(bot, callback, db, user)
	case "masters":
		showMasterStatistics(bot, callback, db, user)
	case "period":
		if len(parts) >= 3 {
			period := models.StatisticsPeriod(parts[2])
			showPeriodStatistics(bot, callback, db, user, period)
		}
	case "refresh":
		// Повторно показываем главное меню статистики
		showStatisticsMainMenu(bot, callback, user)
	default:
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неизвестное действие",
			"uz": "Noma'lum harakat",
			"en": "Unknown action",
		}, lang))
	}
}

// showStatisticsMainMenu показывает главное меню статистики
func showStatisticsMainMenu(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *models.User) {
	lang := user.GetLanguage()

	messageText := "📊 " + i18n.GetText(map[string]string{
		"ru": "Статистика и аналитика",
		"uz": "Statistika va analitika",
		"en": "Statistics and analytics",
	}, lang) + "\n\n" + i18n.GetText(map[string]string{
		"ru": "Выберите тип статистики или период для просмотра:",
		"uz": "Statistika turini yoki ko'rish uchun davrni tanlang:",
		"en": "Choose statistics type or period to view:",
	}, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getStatisticsMainKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing statistics menu text: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing statistics menu markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleDeviceStatusChangeMenu показывает меню выбора нового статуса
func handleDeviceStatusChangeMenu(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, deviceID uint, user *models.User) {
	lang := user.GetLanguage()

	// Получаем устройство
	var device models.Device
	err := db.Where("id = ?", deviceID).First(&device).Error
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Устройство не найдено",
			"uz": "❌ Qurilma topilmadi",
			"en": "❌ Device not found",
		}, lang))
		return
	}

	// Проверяем права доступа
	if user.Role != models.UserRoleAdmin && (device.MasterID == nil || *device.MasterID != user.ID) {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Нет доступа",
			"uz": "❌ Ruxsat yo'q",
			"en": "❌ Access denied",
		}, lang))
		return
	}

	messageText := fmt.Sprintf("%s #%d\n\n%s",
		i18n.GetText(map[string]string{
			"ru": "🔄 Изменение статуса заказа",
			"uz": "🔄 Buyurtma statusini o'zgartirish",
			"en": "🔄 Changing order status",
		}, lang), device.ID,
		i18n.GetText(map[string]string{
			"ru": "Текущий статус: " + getDeviceStatusText(device.Status, lang) + "\n\nВыберите новый статус:",
			"uz": "Joriy status: " + getDeviceStatusText(device.Status, lang) + "\n\nYangi statusni tanlang:",
			"en": "Current status: " + getDeviceStatusText(device.Status, lang) + "\n\nSelect new status:",
		}, lang))

	// Создаем клавиатуру с вариантами статусов
	keyboard := getStatusSelectionKeyboard(deviceID, lang)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending status change menu: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleDeviceStatusSet изменяет статус устройства
func handleDeviceStatusSet(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, deviceID uint, newStatusStr string, user *models.User) {
	lang := user.GetLanguage()

	// Отладочная информация
	log.Printf("handleDeviceStatusSet called: deviceID=%d, newStatusStr='%s', userID=%d", deviceID, newStatusStr, user.TelegramID)

	// Преобразуем строку в DeviceStatus
	newStatus := models.DeviceStatus(newStatusStr)
	if !newStatus.IsValid() {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Неверный статус",
			"uz": "❌ Noto'g'ri status",
			"en": "❌ Invalid status",
		}, lang))
		return
	}

	// Получаем устройство
	var device models.Device
	err := db.Where("id = ?", deviceID).First(&device).Error
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Устройство не найдено",
			"uz": "❌ Qurilma topilmadi",
			"en": "❌ Device not found",
		}, lang))
		return
	}

	// Проверяем права доступа
	if user.Role != models.UserRoleAdmin && (device.MasterID == nil || *device.MasterID != user.ID) {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Нет доступа",
			"uz": "❌ Ruxsat yo'q",
			"en": "❌ Access denied",
		}, lang))
		return
	}

	// Обновляем статус
	oldStatus := device.Status
	device.Status = newStatus

	// Устанавливаем CompletedAt если статус стал "completed"
	if newStatus == models.DeviceStatusCompleted && oldStatus != models.DeviceStatusCompleted {
		now := time.Now()
		device.CompletedAt = &now
	}

	err = db.Save(&device).Error
	if err != nil {
		log.Printf("Error updating device status: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка обновления",
			"uz": "❌ Yangilashda xatolik",
			"en": "❌ Error updating",
		}, lang))
		return
	}

	// Отправляем подтверждение
	messageText := fmt.Sprintf("%s\n\n✅ %s: %s",
		i18n.GetText(map[string]string{
			"ru": "✅ Статус успешно изменен!",
			"uz": "✅ Status muvaffaqiyatli o'zgartirildi!",
			"en": "✅ Status changed successfully!",
		}, lang),
		i18n.GetText(map[string]string{
			"ru": "Новый статус",
			"uz": "Yangi status",
			"en": "New status",
		}, lang), getDeviceStatusText(newStatus, lang))

	// Кнопки для возврата
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🔍 К деталям",
					"uz": "🔍 Tafsilotlarga",
					"en": "🔍 To details",
				}, lang),
				fmt.Sprintf("device_details_%d", deviceID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🔙 К списку заказов",
					"uz": "🔙 Buyurtmalar ro'yxatiga",
					"en": "🔙 To orders list",
				}, lang),
				"devices_refresh"),
		),
	)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending status update confirmation: %v", err)
	}

	// Логируем действие
	log.Printf("User %s (ID: %d) changed status of device %d from %s to %s", user.Name, user.TelegramID, deviceID, oldStatus, newStatus)

	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "✅ Статус изменен",
		"uz": "✅ Status o'zgartirildi",
		"en": "✅ Status changed",
	}, lang))
}

// showGeneralStatistics показывает общую статистику
func showGeneralStatistics(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	statsService := services.NewStatisticsService(db)

	stats, err := statsService.GetOverallStatistics()
	if err != nil {
		log.Printf("Error getting general statistics: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения статистики",
			"uz": "Statistika olishda xatolik",
			"en": "Error getting statistics",
		}, lang))
		return
	}

	messageText := formatGeneralStatistics(stats, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getStatisticsBackKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing general statistics: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing general statistics markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// showMasterStatistics показывает статистику по мастерам
func showMasterStatistics(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	statsService := services.NewStatisticsService(db)

	masterStats, err := statsService.GetMasterStatistics(models.PeriodAllTime)
	if err != nil {
		log.Printf("Error getting master statistics: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения статистики мастеров",
			"uz": "Ustalar statistikasini olishda xatolik",
			"en": "Error getting master statistics",
		}, lang))
		return
	}

	messageText := statsService.FormatMasterStatistics(masterStats, lang)
	if messageText == "👨‍🔧 **Статистика мастеров:**\n\n" {
		messageText = i18n.GetText(map[string]string{
			"ru": "👨‍🔧 Статистика мастеров\n\nПока нет данных для отображения",
			"uz": "👨‍🔧 Ustalar statistikasi\n\nKo'rsatish uchun ma'lumotlar yo'q",
			"en": "👨‍🔧 Master statistics\n\nNo data to display yet",
		}, lang)
	}

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getStatisticsBackKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing master statistics: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing master statistics markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// showPeriodStatistics показывает статистику за период
func showPeriodStatistics(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, db *gorm.DB, user *models.User, period models.StatisticsPeriod) {
	lang := user.GetLanguage()
	statsService := services.NewStatisticsService(db)

	stats, err := statsService.GetPeriodStatistics(period)
	if err != nil {
		log.Printf("Error getting period statistics: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения статистики",
			"uz": "Statistika olishda xatolik",
			"en": "Error getting statistics",
		}, lang))
		return
	}

	messageText := statsService.FormatStatistics(stats, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getStatisticsBackKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing period statistics: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing period statistics markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleBackToStatistics возвращает к главному меню статистики
func handleBackToStatistics(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	// Очищаем состояние пользователя при возврате к статистике
	clearUserState(db, user.TelegramID)

	showStatisticsMainMenu(bot, callback, user)
}

// clearUserState очищает состояние пользователя (вспомогательная функция)
func clearUserState(db *gorm.DB, userTelegramID int64) {
	stateService := services.NewStateService(db)
	stateService.ClearState(userTelegramID)
}

// handleMainMenuCallback обрабатывает возврат в главное меню
func handleMainMenuCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	// Очищаем состояние пользователя при возврате в главное меню
	clearUserState(db, user.TelegramID)
	lang := user.GetLanguage()

	// Редактируем текущее сообщение на главное меню с inline клавиатурой
	messageText := i18n.GetText(map[string]string{
		"ru": "🏠 Главное меню\n\nВыберите действие:",
		"uz": "🏠 Asosiy menyu\n\nAmalni tanlang:",
		"en": "🏠 Main menu\n\nChoose an action:",
	}, lang)

	keyboard := getMainInlineKeyboard(user.Role == models.UserRoleAdmin, lang)

	// Пытаемся отредактировать сообщение
	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMsg.ReplyMarkup = &keyboard

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing main menu text: %v", err)
		// Если не удалось отредактировать текст, отправляем новое сообщение
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
		msg.ReplyMarkup = keyboard
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending new main menu message: %v", err)
		}
	} else {
		// Если редактирование текста прошло успешно, обновляем клавиатуру
		editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, keyboard)
		if _, err := bot.Send(editMarkup); err != nil {
			log.Printf("Error editing main menu markup: %v", err)
		}
	}

	// Отвечаем на callback
	answerCallback(bot, callback.ID, "")
}

// Callback-обработчики для кнопок главного меню

func handleAllOrdersCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	// Создаем фейковый update для совместимости с существующей функцией
	update := tgbotapi.Update{
		CallbackQuery: callback,
	}
	// Создаем фейковый Message
	update.Message = &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: callback.Message.Chat.ID},
		From: callback.From,
	}

	langCache := i18n.NewLanguageCache()
	langCache.Set(callback.From.ID, user.GetLanguage())
	AllOrders(bot, update, cfg, db, langCache)
	answerCallback(bot, callback.ID, "")
}

func handleMyOrdersCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	update := tgbotapi.Update{
		CallbackQuery: callback,
	}
	update.Message = &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: callback.Message.Chat.ID},
		From: callback.From,
	}

	langCache := i18n.NewLanguageCache()
	langCache.Set(callback.From.ID, user.GetLanguage())
	MyOrders(bot, update, cfg, db, langCache)
	answerCallback(bot, callback.ID, "")
}

func handleNewOrderCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	// Отправляем сообщение о создании нового заказа
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "📋 Для создания нового заказа используйте команду /new_order или напишите 'Новый заказ'",
		"uz": "📋 Yangi buyurtma yaratish uchun /new_order buyrug'ini yoki 'Yangi buyurtma' deb yozing",
		"en": "📋 To create a new order, use the /new_order command or write 'New order'",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending new order message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleMastersMenuCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	update := tgbotapi.Update{
		CallbackQuery: callback,
	}
	update.Message = &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: callback.Message.Chat.ID},
		From: callback.From,
	}

	langCache := i18n.NewLanguageCache()
	langCache.Set(callback.From.ID, user.GetLanguage())
	Masters(bot, update, cfg, db, langCache)
	answerCallback(bot, callback.ID, "")
}

func handleAnalyticsCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	update := tgbotapi.Update{
		CallbackQuery: callback,
	}
	update.Message = &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: callback.Message.Chat.ID},
		From: callback.From,
	}

	langCache := i18n.NewLanguageCache()
	langCache.Set(callback.From.ID, user.GetLanguage())
	Analytics(bot, update, cfg, db, langCache)
	answerCallback(bot, callback.ID, "")
}

func handleSearchCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	// Отправляем сообщение о поиске
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "🔍 Для поиска используйте команду /search или напишите 'Поиск'",
		"uz": "🔍 Qidirish uchun /search buyrug'ini yoki 'Qidirish' deb yozing",
		"en": "🔍 To search, use the /search command or write 'Search'",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending search message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

// Недостающие callback-обработчики

func handleBackToOrdersCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	// Очищаем состояние пользователя при возврате к заказам
	clearUserState(db, user.TelegramID)

	if utils.IsAdmin(callback.From.ID, cfg) {
		handleAllOrdersCallback(bot, callback, cfg, db, user)
	} else {
		handleMyOrdersCallback(bot, callback, cfg, db, user)
	}
}

func handleStatusSetCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, status string) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "⚠️ Функция изменения статуса находится в разработке",
		"uz": "⚠️ Status o'zgartirish funksiyasi ishlab chiqilmoqda",
		"en": "⚠️ Status change function is under development",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending status message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleHelpContactCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *models.User) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "📞 Для связи с поддержкой обратитесь к администратору",
		"uz": "📞 Qo'llab-quvvatlash uchun administratorga murojaat qiling",
		"en": "📞 To contact support, please contact the administrator",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending contact message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleOrdersRefreshCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	handleBackToOrdersCallback(bot, callback, cfg, db, user)
}

func handleChangeStatusCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "🔄 Функция изменения статуса заказа находится в разработке",
		"uz": "🔄 Buyurtma statusini o'zgartirish funksiyasi ishlab chiqilmoqda",
		"en": "🔄 Order status change function is under development",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending change status message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleSetPriceCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "💰 Функция установки цены находится в разработке",
		"uz": "💰 Narx belgilash funksiyasi ishlab chiqilmoqda",
		"en": "💰 Price setting function is under development",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending set price message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleAssignMasterCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "👨‍🔧 Функция назначения мастера находится в разработке",
		"uz": "👨‍🔧 Usta tayinlash funksiyasi ishlab chiqilmoqda",
		"en": "👨‍🔧 Master assignment function is under development",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending assign master message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleEditOrderCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "✏️ Функция редактирования заказа находится в разработке",
		"uz": "✏️ Buyurtmani tahrirlash funksiyasi ishlab chiqilmoqda",
		"en": "✏️ Order editing function is under development",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending edit order message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleOrderDetailsCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "ℹ️ Функция просмотра деталей заказа находится в разработке",
		"uz": "ℹ️ Buyurtma tafsilotlarini ko'rish funksiyasi ishlab chiqilmoqda",
		"en": "ℹ️ Order details viewing function is under development",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending order details message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleOrdersPageCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, i18n.GetText(map[string]string{
		"ru": "📄 Функция пагинации находится в разработке",
		"uz": "📄 Sahifalash funksiyasi ishlab chiqilmoqda",
		"en": "📄 Pagination function is under development",
	}, lang))

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending pagination message: %v", err)
	}
	answerCallback(bot, callback.ID, "")
}

func handleStatusSetCallbackComposite(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	// Парсим status_set_{orderID}_{status}
	parts := strings.Split(action, "_")
	if len(parts) >= 3 {
		handleStatusSetCallback(bot, callback, cfg, db, user, parts[2])
	} else {
		handleStatusSetCallback(bot, callback, cfg, db, user, "unknown")
	}
}

func handleMasterActionCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	// Парсим master_action_{action}_{masterID}
	parts := strings.Split(action, "_")
	if len(parts) < 4 {
		answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidDataFormat, user.GetLanguage()))
		return
	}

	masterAction := parts[2]
	masterID := parts[3]

	switch masterAction {
	case "profile":
		// Показать профиль мастера
		handleMasterProfileCallback(bot, callback, cfg, db, user, masterID)
	case "toggle":
		// Переключить статус мастера
		handleMasterToggleCallback(bot, callback, cfg, db, user, masterID)
	case "orders":
		// Показать заказы мастера
		handleMasterOrdersCallback(bot, callback, cfg, db, user, masterID)
	case "refresh":
		// Обновить профиль мастера
		handleMasterProfileCallback(bot, callback, cfg, db, user, masterID)
	default:
		answerCallback(bot, callback.ID, i18n.GetText(i18n.UnknownAction, user.GetLanguage()))
	}
}

func handleMasterProfileCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, masterID string) {
	lang := user.GetLanguage()

	// Конвертируем masterID в uint
	id, err := strconv.ParseUint(masterID, 10, 32)
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный ID мастера",
			"uz": "Noto'g'ri usta ID",
			"en": "Invalid master ID",
		}, lang))
		return
	}

	masterService := services.NewMasterService(db)
	masterInfo, err := masterService.GetMasterWithStats(uint(id))
	if err != nil {
		log.Printf("Error getting master profile: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения профиля мастера",
			"uz": "Usta profilini olishda xatolik",
			"en": "Error getting master profile",
		}, lang))
		return
	}

	// Форматируем профиль мастера
	messageText := masterService.FormatMasterProfile(masterInfo, lang)

	// Создаем клавиатуру
	keyboard := getMasterProfileKeyboard(uint(id), masterInfo.User.IsActive, lang)

	// Редактируем сообщение
	editText := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editText.ParseMode = "Markdown"
	if _, err := bot.Send(editText); err != nil {
		log.Printf("Error editing master profile text: %v", err)
	}

	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, keyboard)
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing master profile markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

func handleMasterToggleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, masterID string) {
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

	// Конвертируем masterID в uint
	id, err := strconv.ParseUint(masterID, 10, 32)
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный ID мастера",
			"uz": "Noto'g'ri usta ID",
			"en": "Invalid master ID",
		}, lang))
		return
	}

	masterService := services.NewMasterService(db)

	// Переключаем статус
	err = masterService.ToggleMasterStatus(uint(id))
	if err != nil {
		log.Printf("Error toggling master status: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка изменения статуса мастера",
			"uz": "Usta statusini o'zgartirishda xatolik",
			"en": "Error changing master status",
		}, lang))
		return
	}

	// Получаем обновлённые данные мастера
	masterInfo, err := masterService.GetMasterWithStats(uint(id))
	if err != nil {
		log.Printf("Error getting updated master info: %v", err)
		return
	}

	// Обновляем отображение профиля
	messageText := masterService.FormatMasterProfile(masterInfo, lang)
	keyboard := getMasterProfileKeyboard(uint(id), masterInfo.User.IsActive, lang)

	// Редактируем сообщение
	editText := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editText.ParseMode = "Markdown"
	if _, err := bot.Send(editText); err != nil {
		log.Printf("Error editing master profile after toggle: %v", err)
	}

	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, keyboard)
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing master profile markup after toggle: %v", err)
	}

	// Подтверждаем успешное изменение
	statusText := "Активирован"
	if !masterInfo.User.IsActive {
		statusText = "Деактивирован"
	}

	switch lang {
	case "uz":
		if masterInfo.User.IsActive {
			statusText = "Faollashtirildi"
		} else {
			statusText = "Faolsizlantirildi"
		}
	case "en":
		if masterInfo.User.IsActive {
			statusText = "Activated"
		} else {
			statusText = "Deactivated"
		}
	}

	answerCallback(bot, callback.ID, fmt.Sprintf("✅ %s", statusText))
}

func handleMasterOrdersCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, masterID string) {
	lang := user.GetLanguage()

	// Конвертируем masterID в uint
	id, err := strconv.ParseUint(masterID, 10, 32)
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный ID мастера",
			"uz": "Noto'g'ri usta ID",
			"en": "Invalid master ID",
		}, lang))
		return
	}

	masterService := services.NewMasterService(db)

	// Получаем заказы мастера
	devices, err := masterService.GetMasterDevices(uint(id), "")
	if err != nil {
		log.Printf("Error getting master devices: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения заказов мастера",
			"uz": "Usta buyurtmalarini olishda xatolik",
			"en": "Error getting master orders",
		}, lang))
		return
	}

	// Получаем информацию о мастере
	masterInfo, err := masterService.GetMasterWithStats(uint(id))
	if err != nil {
		log.Printf("Error getting master info: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения данных мастера",
			"uz": "Usta ma'lumotlarini olishda xatolik",
			"en": "Error getting master data",
		}, lang))
		return
	}

	var messageText string
	if len(devices) == 0 {
		messageText = fmt.Sprintf("📋 %s\n\n%s: %s\n\n%s",
			i18n.GetText(map[string]string{
				"ru": "Заказы мастера",
				"uz": "Usta buyurtmalari",
				"en": "Master orders",
			}, lang),
			i18n.GetText(map[string]string{
				"ru": "Мастер",
				"uz": "Usta",
				"en": "Master",
			}, lang),
			masterInfo.User.FullName(),
			i18n.GetText(map[string]string{
				"ru": "У мастера пока нет заказов",
				"uz": "Ustada hozircha buyurtmalar yo'q",
				"en": "The master has no orders yet",
			}, lang))
	} else {
		// Форматируем список заказов
		messageText = fmt.Sprintf("📋 %s\n\n%s: **%s**\n\n",
			i18n.GetText(map[string]string{
				"ru": "Заказы мастера",
				"uz": "Usta buyurtmalari",
				"en": "Master orders",
			}, lang),
			i18n.GetText(map[string]string{
				"ru": "Мастер",
				"uz": "Usta",
				"en": "Master",
			}, lang),
			masterInfo.User.FullName())

		// Показываем только первые 5 заказов
		maxOrders := len(devices)
		if maxOrders > 5 {
			maxOrders = 5
		}

		for i := 0; i < maxOrders; i++ {
			device := devices[i]
			statusText := device.Status.Text()

			messageText += fmt.Sprintf("**%d. %s %s**\n",
				i+1, device.DeviceType, device.Brand)

			if device.Model != "" {
				messageText += fmt.Sprintf("📱 %s\n", device.Model)
			}

			messageText += fmt.Sprintf("⚡️ %s: %s\n",
				i18n.GetText(map[string]string{
					"ru": "Статус",
					"uz": "Status",
					"en": "Status",
				}, lang), statusText)

			if device.Customer != nil {
				messageText += fmt.Sprintf("👤 %s: %s\n",
					i18n.GetText(map[string]string{
						"ru": "Клиент",
						"uz": "Mijoz",
						"en": "Customer",
					}, lang), device.Customer.Name)
			}

			if device.TotalCost > 0 {
				messageText += fmt.Sprintf("💰 %.0f сум\n", device.TotalCost)
			}

			messageText += "\n"
		}

		if len(devices) > 5 {
			messageText += fmt.Sprintf("… и ещё %d заказов", len(devices)-5)
		}
	}

	// Создаём клавиатуру с возвратом
	var keyboard [][]tgbotapi.InlineKeyboardButton

	// Кнопка возврата к профилю мастера
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			i18n.GetText(map[string]string{
				"ru": "👤 К профилю мастера",
				"uz": "👤 Usta profiliga",
				"en": "👤 To master profile",
			}, lang),
			fmt.Sprintf("master_action_profile_%s", masterID)),
	})

	// Кнопка обновления
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			i18n.GetText(map[string]string{
				"ru": "🔄 Обновить",
				"uz": "🔄 Yangilash",
				"en": "🔄 Refresh",
			}, lang),
			fmt.Sprintf("master_action_orders_%s", masterID)),
	})

	// Кнопка возврата к списку мастеров
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			i18n.GetText(map[string]string{
				"ru": "👥 К мастерам",
				"uz": "👥 Ustalarga",
				"en": "👥 To masters",
			}, lang),
			"back_to_masters"),
	})

	// Редактируем сообщение
	editText := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editText.ParseMode = "Markdown"
	if _, err := bot.Send(editText); err != nil {
		log.Printf("Error editing master orders text: %v", err)
	}

	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing master orders markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// formatGeneralStatistics форматирует общую статистику для отображения
func formatGeneralStatistics(stats *models.OrderStatistics, lang string) string {
	var result strings.Builder

	result.WriteString("📊 " + i18n.GetText(map[string]string{
		"ru": "Общая статистика",
		"uz": "Umumiy statistika",
		"en": "General statistics",
	}, lang) + "\n\n")

	result.WriteString("📈 " + i18n.GetText(map[string]string{
		"ru": "Основные показатели:",
		"uz": "Asosiy ko'rsatkichlar:",
		"en": "Main indicators:",
	}, lang) + "\n")

	result.WriteString(fmt.Sprintf("• %s: %d\n",
		i18n.GetText(map[string]string{
			"ru": "Всего заказов",
			"uz": "Jami buyurtmalar",
			"en": "Total orders",
		}, lang), stats.TotalOrders))

	result.WriteString(fmt.Sprintf("• %s: %.2f сум\n",
		i18n.GetText(map[string]string{
			"ru": "Общая выручка",
			"uz": "Umumiy daromad",
			"en": "Total revenue",
		}, lang), stats.TotalRevenue))

	if stats.AverageRepairTime > 0 {
		result.WriteString(fmt.Sprintf("• %s: %.1f дней\n",
			i18n.GetText(map[string]string{
				"ru": "Среднее время ремонта",
				"uz": "O'rtacha ta'mirlash vaqti",
				"en": "Average repair time",
			}, lang), stats.AverageRepairTime))
	}
	result.WriteString("\n")

	// Статистика по статусам
	if len(stats.OrdersByStatus) > 0 {
		result.WriteString("🔄 " + i18n.GetText(map[string]string{
			"ru": "По статусам:",
			"uz": "Statuslar bo'yicha:",
			"en": "By status:",
		}, lang) + "\n")

		for status, count := range stats.OrdersByStatus {
			result.WriteString(fmt.Sprintf("• %s: %d\n", status.Text(), count))
		}
		result.WriteString("\n")
	}

	// Топ брендов
	if len(stats.TopBrands) > 0 {
		result.WriteString("🏷️ " + i18n.GetText(map[string]string{
			"ru": "Популярные бренды:",
			"uz": "Mashur brendlar:",
			"en": "Popular brands:",
		}, lang) + "\n")

		for brand, count := range stats.TopBrands {
			result.WriteString(fmt.Sprintf("• %s: %d\n", brand, count))
		}
	}

	return result.String()
}

// handleMastersCallback обрабатывает callback'и управления мастерами
func handleMastersCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, parts []string, user *models.User) {
	lang := user.GetLanguage()

	if len(parts) < 2 {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный формат данных",
			"uz": "Noto'g'ri ma'lumot formati",
			"en": "Invalid data format",
		}, lang))
		return
	}

	masterService := services.NewMasterService(db)
	action := parts[1]

	switch action {
	case "list":
		if len(parts) >= 3 && parts[2] == "all" {
			showMastersList(bot, callback, masterService, user, false)
		} else if len(parts) >= 3 && parts[2] == "active" {
			showMastersList(bot, callback, masterService, user, true)
		}
	case "add":
		showAddMasterForm(bot, callback, user, db)
	case "refresh":
		showMastersMainMenu(bot, callback, user)
	default:
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неизвестное действие",
			"uz": "Noma'lum harakat",
			"en": "Unknown action",
		}, lang))
	}
}

// handleMasterCallback обрабатывает callback'и конкретного мастера
func handleMasterCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, parts []string, user *models.User) {
	lang := user.GetLanguage()

	if len(parts) < 3 {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный формат данных",
			"uz": "Noto'g'ri ma'lumot formati",
			"en": "Invalid data format",
		}, lang))
		return
	}

	masterService := services.NewMasterService(db)
	action := parts[1]

	switch action {
	case "profile":
		if len(parts) >= 3 {
			masterID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
					"ru": "Неверный ID мастера",
					"uz": "Noto'g'ri usta ID",
					"en": "Invalid master ID",
				}, lang))
				return
			}
			showMasterProfile(bot, callback, masterService, user, uint(masterID))
		}
	case "toggle":
		if len(parts) >= 3 {
			masterID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
					"ru": "Неверный ID мастера",
					"uz": "Noto'g'ri usta ID",
					"en": "Invalid master ID",
				}, lang))
				return
			}
			toggleMasterStatus(bot, callback, masterService, user, uint(masterID))
		}
	case "refresh":
		if len(parts) >= 3 {
			masterID, err := strconv.ParseUint(parts[2], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
					"ru": "Неверный ID мастера",
					"uz": "Noto'g'ri usta ID",
					"en": "Invalid master ID",
				}, lang))
				return
			}
			showMasterProfile(bot, callback, masterService, user, uint(masterID))
		}
	default:
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неизвестное действие",
			"uz": "Noma'lum harakat",
			"en": "Unknown action",
		}, lang))
	}
}

// handleAssignCallback обрабатывает назначение мастера на заказ
func handleAssignCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, parts []string, user *models.User) {
	lang := user.GetLanguage()

	if len(parts) < 4 || parts[1] != "master" {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный формат данных",
			"uz": "Noto'g'ri ma'lumot formati",
			"en": "Invalid data format",
		}, lang))
		return
	}

	deviceID, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный ID заказа",
			"uz": "Noto'g'ri buyurtma ID",
			"en": "Invalid order ID",
		}, lang))
		return
	}

	masterID, err := strconv.ParseUint(parts[3], 10, 32)
	if err != nil {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Неверный ID мастера",
			"uz": "Noto'g'ri usta ID",
			"en": "Invalid master ID",
		}, lang))
		return
	}

	masterService := services.NewMasterService(db)
	err = masterService.AssignMasterToDevice(uint(deviceID), uint(masterID))
	if err != nil {
		log.Printf("Error assigning master to device: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка назначения мастера",
			"uz": "Usta tayinlashda xatolik",
			"en": "Error assigning master",
		}, lang))
		return
	}

	// Получаем информацию о мастере для уведомления
	master, err := masterService.GetMasterByID(uint(masterID))
	if err != nil {
		log.Printf("Error getting master info: %v", err)
	}

	successText := i18n.GetText(map[string]string{
		"ru": "✅ Мастер назначен на заказ",
		"uz": "✅ Usta buyurtmaga tayinlandi",
		"en": "✅ Master assigned to order",
	}, lang)

	if master != nil {
		successText = fmt.Sprintf(i18n.GetText(map[string]string{
			"ru": "✅ Мастер %s назначен на заказ",
			"uz": "✅ Usta %s buyurtmaga tayinlandi",
			"en": "✅ Master %s assigned to order",
		}, lang), master.FullName())
	}

	answerCallback(bot, callback.ID, successText)

	// Обновляем сообщение с заказом (возвращаемся к списку заказов)
	// TODO: Здесь можно добавить обновление конкретного заказа
}

// showMastersMainMenu показывает главное меню управления мастерами
func showMastersMainMenu(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *models.User) {
	lang := user.GetLanguage()

	messageText := "👥 " + i18n.GetText(map[string]string{
		"ru": "Управление мастерами",
		"uz": "Ustalarni boshqarish",
		"en": "Masters management",
	}, lang) + "\n\n" + i18n.GetText(map[string]string{
		"ru": "Выберите действие для управления мастерами:",
		"uz": "Ustalarni boshqarish uchun amalni tanlang:",
		"en": "Choose an action for masters management:",
	}, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getMastersMainKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing masters menu text: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing masters menu markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// showMastersList показывает список мастеров
func showMastersList(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, masterService *services.MasterService, user *models.User, activeOnly bool) {
	lang := user.GetLanguage()

	var masters []models.User
	var err error

	if activeOnly {
		masters, err = masterService.GetActiveMasters()
	} else {
		masters, err = masterService.GetAllMasters()
	}

	if err != nil {
		log.Printf("Error getting masters list: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения списка мастеров",
			"uz": "Ustalar ro'yxatini olishda xatolik",
			"en": "Error getting masters list",
		}, lang))
		return
	}

	messageText := masterService.FormatMastersList(masters, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getMasterListKeyboard(masters, lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing masters list: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing masters list markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// showMasterProfile показывает профиль мастера
func showMasterProfile(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, masterService *services.MasterService, user *models.User, masterID uint) {
	lang := user.GetLanguage()

	info, err := masterService.GetMasterWithStats(masterID)
	if err != nil {
		log.Printf("Error getting master profile: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка получения профиля мастера",
			"uz": "Usta profilini olishda xatolik",
			"en": "Error getting master profile",
		}, lang))
		return
	}

	messageText := masterService.FormatMasterProfile(info, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getMasterProfileKeyboard(masterID, info.User.IsActive, lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing master profile: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing master profile markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// toggleMasterStatus переключает статус активности мастера
func toggleMasterStatus(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, masterService *services.MasterService, user *models.User, masterID uint) {
	lang := user.GetLanguage()

	err := masterService.ToggleMasterStatus(masterID)
	if err != nil {
		log.Printf("Error toggling master status: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "Ошибка изменения статуса мастера",
			"uz": "Usta holatini o'zgartirishda xatolik",
			"en": "Error changing master status",
		}, lang))
		return
	}

	answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
		"ru": "✅ Статус мастера изменен",
		"uz": "✅ Usta holati o'zgartirildi",
		"en": "✅ Master status changed",
	}, lang))

	// Обновляем профиль мастера
	showMasterProfile(bot, callback, masterService, user, masterID)
}

// handleBackToMasters возвращает к главному меню мастеров
func handleBackToMasters(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	// Очищаем состояние пользователя при возврате к мастерам
	clearUserState(db, user.TelegramID)

	showMastersMainMenu(bot, callback, user)
}

// showAddMasterForm показывает форму добавления мастера
func showAddMasterForm(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, user *models.User, db *gorm.DB) {
	lang := user.GetLanguage()
	stateService := services.NewStateService(db)

	// Устанавливаем состояние ожидания Telegram ID
	stateService.SetState(user.TelegramID, models.StateWaitingMasterTelegramID, nil)

	messageText := i18n.GetText(map[string]string{
		"ru": "➕ Добавление нового мастера\n\nВведите Telegram ID пользователя, которого хотите сделать мастером:\n\nПример: 123456789\n\nℹ️ Пользователь должен сначала написать боту /start",
		"uz": "➕ Yangi usta qo'shish\n\nUsta qilmoqchi bo'lgan foydalanuvchining Telegram ID raqamini kiriting:\n\nMisol: 123456789\n\nℹ️ Foydalanuvchi avval botga /start yozgan bo'lishi kerak",
		"en": "➕ Adding new master\n\nEnter the Telegram ID of the user you want to make a master:\n\nExample: 123456789\n\nℹ️ The user must first write /start to the bot",
	}, lang)

	// Кнопка отмены
	cancelButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "❌ Отмена",
					"uz": "❌ Bekor qilish",
					"en": "❌ Cancel",
				}, lang),
				"back_to_masters"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, cancelButton)

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing add master form text: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing add master form markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleMasterTelegramIDInput обрабатывает ввод Telegram ID для добавления мастера
func handleMasterTelegramIDInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB, stateService *services.StateService, user *models.User, inputText, lang string) bool {
	// Проверяем, что введено число
	telegramID, err := strconv.ParseInt(inputText, 10, 64)
	if err != nil || telegramID <= 0 {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Неверный формат Telegram ID. Введите числовой ID.\n\nПример: 123456789",
			"uz": "❌ Telegram ID formati noto'g'ri. Raqamli ID kiriting.\n\nMisol: 123456789",
			"en": "❌ Invalid Telegram ID format. Enter numeric ID.\n\nExample: 123456789",
		}, lang))
		return true
	}

	// Проверяем, существует ли пользователь
	var targetUser models.User
	result := db.Where("telegram_id = ?", telegramID).First(&targetUser)

	if result.Error == gorm.ErrRecordNotFound {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Пользователь с таким Telegram ID не найден.\n\nПользователь должен сначала написать боту /start",
			"uz": "❌ Bunday Telegram ID li foydalanuvchi topilmadi.\n\nFoydalanuvchi avval botga /start yozgan bo'lishi kerak",
			"en": "❌ User with this Telegram ID not found.\n\nThe user must first write /start to the bot",
		}, lang))
		return true
	}

	if result.Error != nil {
		log.Printf("Error finding user: %v", result.Error)
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка при поиске пользователя",
			"uz": "❌ Foydalanuvchini qidirishda xatolik",
			"en": "❌ Error finding user",
		}, lang))
		return true
	}

	// Проверяем, не является ли пользователь уже мастером или админом
	if targetUser.Role == models.UserRoleMaster {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": fmt.Sprintf("ℹ️ Пользователь %s уже является мастером", targetUser.FullName()),
			"uz": fmt.Sprintf("ℹ️ Foydalanuvchi %s allaqachon usta", targetUser.FullName()),
			"en": fmt.Sprintf("ℹ️ User %s is already a master", targetUser.FullName()),
		}, lang))
		stateService.ClearState(user.TelegramID)
		return true
	}

	if targetUser.Role == models.UserRoleAdmin {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": fmt.Sprintf("❌ Пользователь %s является администратором. Нельзя изменить роль", targetUser.FullName()),
			"uz": fmt.Sprintf("❌ Foydalanuvchi %s administrator. Rolni o'zgartirib bo'lmaydi", targetUser.FullName()),
			"en": fmt.Sprintf("❌ User %s is an administrator. Cannot change role", targetUser.FullName()),
		}, lang))
		stateService.ClearState(user.TelegramID)
		return true
	}

	// Изменяем роль на мастера
	targetUser.Role = models.UserRoleMaster
	targetUser.IsActive = true

	if err := db.Save(&targetUser).Error; err != nil {
		log.Printf("Error updating user role: %v", err)
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка при изменении роли пользователя",
			"uz": "❌ Foydalanuvchi rolini o'zgartirishda xatolik",
			"en": "❌ Error changing user role",
		}, lang))
		return true
	}

	// Отправляем уведомление об успехе
	successText := fmt.Sprintf(i18n.GetText(map[string]string{
		"ru": "✅ Пользователь %s успешно назначен мастером!\n\n👤 Имя: %s\n🆔 ID: %d\n📱 Telegram ID: %d",
		"uz": "✅ Foydalanuvchi %s muvaffaqiyatli usta etib tayinlandi!\n\n👤 Ismi: %s\n🆔 ID: %d\n📱 Telegram ID: %d",
		"en": "✅ User %s successfully assigned as master!\n\n👤 Name: %s\n🆔 ID: %d\n📱 Telegram ID: %d",
	}, lang), targetUser.FullName(), targetUser.FullName(), targetUser.ID, targetUser.TelegramID)

	// Кнопка возврата к мастерам
	backButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "👥 К списку мастеров",
					"uz": "👥 Ustalar ro'yxatiga",
					"en": "👥 To masters list",
				}, lang),
				"masters_list_all"),
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🏠 Главное меню",
					"uz": "🏠 Asosiy menyu",
					"en": "🏠 Main menu",
				}, lang),
				"main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, successText)
	msg.ReplyMarkup = backButton

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending success message: %v", err)
	}

	// Уведомляем нового мастера
	notifyNewMaster(bot, &targetUser)

	// Очищаем состояние
	stateService.ClearState(user.TelegramID)

	log.Printf("User %d (%s) promoted to master by admin %d", targetUser.TelegramID, targetUser.FullName(), user.TelegramID)
	return true
}

// handleStatsGeneralCallback обрабатывает callback общей статистики
func handleStatsGeneralCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	statsService := services.NewStatisticsService(db)

	stats, err := statsService.GetOverallStatistics()
	if err != nil {
		log.Printf("Error getting general statistics: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка получения статистики",
			"uz": "❌ Statistika olishda xatolik",
			"en": "❌ Error getting statistics",
		}, lang))
		return
	}

	messageText := formatGeneralStatistics(stats, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getStatisticsBackKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing stats general message: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing stats general markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleStatsMastersCallback обрабатывает callback статистики мастеров
func handleStatsMastersCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	statsService := services.NewStatisticsService(db)

	masterStats, err := statsService.GetMasterStatistics(models.PeriodAllTime)
	if err != nil {
		log.Printf("Error getting master statistics: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка получения статистики мастеров",
			"uz": "❌ Ustalar statistikasini olishda xatolik",
			"en": "❌ Error getting master statistics",
		}, lang))
		return
	}

	messageText := statsService.FormatMasterStatistics(masterStats, lang)
	if len(masterStats) == 0 {
		messageText = i18n.GetText(map[string]string{
			"ru": "👨‍🔧 Статистика мастеров\n\nПока нет данных для отображения",
			"uz": "👨‍🔧 Ustalar statistikasi\n\nKo'rsatish uchun ma'lumotlar yo'q",
			"en": "👨‍🔧 Master statistics\n\nNo data to display yet",
		}, lang)
	}

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getStatisticsBackKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing stats masters message: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing stats masters markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleStatsPeriodCallback обрабатывает callback статистики за период
func handleStatsPeriodCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User, action string) {
	lang := user.GetLanguage()

	// Извлекаем период из action (stats_period_today, stats_period_week, etc.)
	parts := strings.Split(action, "_")
	if len(parts) < 3 {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Неверный формат периода",
			"uz": "❌ Noto'g'ri davr formati",
			"en": "❌ Invalid period format",
		}, lang))
		return
	}

	var period models.StatisticsPeriod
	periodStr := parts[2]

	switch periodStr {
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
	case "all", "time":
		period = models.PeriodAllTime
	default:
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Неизвестный период",
			"uz": "❌ Noma'lum davr",
			"en": "❌ Unknown period",
		}, lang))
		return
	}

	statsService := services.NewStatisticsService(db)
	stats, err := statsService.GetPeriodStatistics(period)
	if err != nil {
		log.Printf("Error getting period statistics: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка получения статистики",
			"uz": "❌ Statistika olishda xatolik",
			"en": "❌ Error getting statistics",
		}, lang))
		return
	}

	messageText := statsService.FormatStatistics(stats, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, getStatisticsBackKeyboard(lang))

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing stats period message: %v", err)
	}
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing stats period markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// notifyNewMaster уведомляет пользователя о назначении мастером
func notifyNewMaster(bot *tgbotapi.BotAPI, user *models.User) {
	lang := user.GetLanguage()

	messageText := i18n.GetText(map[string]string{
		"ru": "🎉 Поздравляем! Вы назначены мастером в ремонтной мастерской!\n\nТеперь у вас есть доступ к управлению заказами. Используйте /start чтобы увидеть новые возможности.",
		"uz": "🎉 Tabriklaymiz! Siz ta'mirlash ustaxonasida usta etib tayinlandingiz!\n\nEndi sizda buyurtmalarni boshqarish imkoniyati bor. Yangi imkoniyatlarni ko'rish uchun /start dan foydalaning.",
		"en": "🎉 Congratulations! You have been appointed as a master in the repair shop!\n\nYou now have access to order management. Use /start to see new features.",
	}, lang)

	msg := tgbotapi.NewMessage(user.TelegramID, messageText)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error notifying new master %d: %v", user.TelegramID, err)
	}
}

// handleMastersListAllCallback обрабатывает показ всех мастеров
func handleMastersListAllCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	masterService := services.NewMasterService(db)

	masters, err := masterService.GetAllMasters()
	if err != nil {
		log.Printf("Error getting all masters: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка получения списка мастеров",
			"uz": "❌ Ustalar ro'yxatini olishda xatolik",
			"en": "❌ Error getting masters list",
		}, lang))
		return
	}

	messageText := masterService.FormatMastersList(masters, lang)
	keyboard := getMasterListKeyboard(masters, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMsg.ParseMode = "Markdown"
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing masters list text: %v", err)
	}

	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, keyboard)
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing masters list markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleMastersListActiveCallback обрабатывает показ активных мастеров
func handleMastersListActiveCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	lang := user.GetLanguage()
	masterService := services.NewMasterService(db)

	masters, err := masterService.GetActiveMasters()
	if err != nil {
		log.Printf("Error getting active masters: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Ошибка получения списка активных мастеров",
			"uz": "❌ Faol ustalar ro'yxatini olishda xatolik",
			"en": "❌ Error getting active masters list",
		}, lang))
		return
	}

	messageText := masterService.FormatMastersList(masters, lang)
	keyboard := getMasterListKeyboard(masters, lang)

	editMsg := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, messageText)
	editMsg.ParseMode = "Markdown"
	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing active masters list text: %v", err)
	}

	editMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, keyboard)
	if _, err := bot.Send(editMarkup); err != nil {
		log.Printf("Error editing active masters list markup: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}

// handleMastersAddCallback обрабатывает добавление нового мастера
func handleMastersAddCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, user *models.User) {
	// Используем существующую функцию showAddMasterForm
	showAddMasterForm(bot, callback, user, db)
}

// handleDeviceDetails показывает детали устройства с кнопками управления для мастеров
func handleDeviceDetails(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, cfg *config.Config, db *gorm.DB, deviceID uint, user *models.User) {
	lang := user.GetLanguage()

	// Получаем устройство из базы данных
	var device models.Device
	err := db.Preload("Customer").Preload("Master").Where("id = ?", deviceID).First(&device).Error
	if err != nil {
		log.Printf("Error getting device details: %v", err)
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Устройство не найдено",
			"uz": "❌ Qurilma topilmadi",
			"en": "❌ Device not found",
		}, lang))
		return
	}

	// Проверяем права доступа
	if user.Role != models.UserRoleAdmin && (device.MasterID == nil || *device.MasterID != user.ID) {
		answerCallback(bot, callback.ID, i18n.GetText(map[string]string{
			"ru": "❌ Нет доступа к этому заказу",
			"uz": "❌ Bu buyurtmaga ruxsat yo'q",
			"en": "❌ No access to this order",
		}, lang))
		return
	}

	// Формируем сообщение с деталями
	messageText := fmt.Sprintf("%s #%d\n\n",
		i18n.GetText(map[string]string{
			"ru": "🔍 Детали заказа",
			"uz": "🔍 Buyurtma tafsilotlari",
			"en": "🔍 Order details",
		}, lang), device.ID)

	messageText += fmt.Sprintf("🆔 %s: `%s`\n",
		i18n.GetText(map[string]string{
			"ru": "Код",
			"uz": "Kod",
			"en": "Code",
		}, lang), device.Code)

	messageText += fmt.Sprintf("📱 %s: %s %s\n",
		i18n.GetText(map[string]string{
			"ru": "Устройство",
			"uz": "Qurilma",
			"en": "Device",
		}, lang), device.Brand, device.Model)

	if device.Customer != nil {
		messageText += fmt.Sprintf("👤 %s: %s\n",
			i18n.GetText(map[string]string{
				"ru": "Клиент",
				"uz": "Mijoz",
				"en": "Customer",
			}, lang), device.Customer.Name)

		if device.Customer.Phone != "" {
			messageText += fmt.Sprintf("📞 %s\n", device.Customer.Phone)
		}
	}

	if device.Master != nil {
		messageText += fmt.Sprintf("🔧 %s: %s\n",
			i18n.GetText(map[string]string{
				"ru": "Мастер",
				"uz": "Usta",
				"en": "Master",
			}, lang), device.Master.FullName())
	}

	messageText += fmt.Sprintf("⚡️ %s: %s\n",
		i18n.GetText(map[string]string{
			"ru": "Статус",
			"uz": "Status",
			"en": "Status",
		}, lang), getDeviceStatusText(device.Status, lang))

	if device.RepairCost > 0 {
		messageText += fmt.Sprintf("💰 %s: %.0f сум\n",
			i18n.GetText(map[string]string{
				"ru": "Стоимость ремонта",
				"uz": "Ta'mirlash narxi",
				"en": "Repair cost",
			}, lang), device.RepairCost)
	}

	if device.Problem != "" {
		messageText += fmt.Sprintf("\n🔍 %s:\n%s\n",
			i18n.GetText(map[string]string{
				"ru": "Описание проблемы",
				"uz": "Muammo tavsifi",
				"en": "Problem description",
			}, lang), device.Problem)
	}

	// Создаем клавиатуру с кнопками управления
	keyboard := getDeviceManagementKeyboard(&device, user, lang)

	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, messageText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending device details: %v", err)
	}

	answerCallback(bot, callback.ID, "")
}
