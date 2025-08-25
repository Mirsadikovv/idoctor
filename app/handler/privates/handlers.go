package handlers

import (
	"log"
	"strconv"
	"strings"

	"idoctor-bot/app/config"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
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
		default:
			Help(bot, update, langCache)
		}
	} else if update.Message != nil {
		// Получаем язык пользователя
		lang := langCache.Get(update.Message.From.ID)
		
		// Обработка смены языка
		if update.Message.Text == i18n.GetButton("russian", lang) || update.Message.Text == i18n.GetButton("uzbek", lang) || update.Message.Text == i18n.GetButton("english", lang) {
			ChangeLanguage(bot, update, cfg, db, langCache)
			return
		}

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
		case i18n.GetButton("menu", lang):
			Menu(bot, update, cfg, db, langCache)
		case i18n.GetButton("change_language", lang):
			ShowLanguageMenu(bot, update, cfg, db, langCache)
		default:
			Help(bot, update, langCache)
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
		if len(parts) >= 2 {
			orderID, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, i18n.GetText(i18n.InvalidOrderID, lang))
				return
			}
			handleOrderDetails(bot, callback, cfg, db, uint(orderID), &user)
		}
	case "status":
		if len(parts) >= 3 {
			orderID, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, "❌ Неверный ID заказа")
				return
			}
			newStatus := models.DeviceStatus(parts[2])
			handleStatusChange(bot, callback, cfg, db, uint(orderID), newStatus, &user)
		}
	case "price":
		if len(parts) >= 2 {
			orderID, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				answerCallback(bot, callback.ID, "❌ Неверный ID заказа")
				return
			}
			handlePriceSet(bot, callback, cfg, db, uint(orderID), &user)
		}
	case "back":
		if len(parts) >= 2 && parts[1] == "orders" {
			handleBackToOrders(bot, callback, cfg, db, &user)
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