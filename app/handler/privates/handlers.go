package handlers

import (
	"log"

	"idoctor-bot/app/config"
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

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	if update.Message == nil {
		return
	}

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			Start(bot, update, cfg, db)
		case "menu":
			Menu(bot, update, cfg, db)
		case "orders":
			MyOrders(bot, update, cfg, db)
		case "help":
			Help(bot, update)
		default:
			Help(bot, update)
		}
	} else if update.Message != nil {
		switch update.Message.Text {
		case "📋 Мои заказы":
			MyOrders(bot, update, cfg, db)
		case "📊 Все заказы":
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				AllOrders(bot, update, cfg, db)
			} else {
				MyOrders(bot, update, cfg, db)
			}
		case "➕ Новый заказ":
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				NewOrder(bot, update, cfg, db)
			} else {
				sendMessage(bot, update.Message.Chat.ID, "❌ У вас нет доступа к этой функции")
			}
		case "👥 Мастера":
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				Masters(bot, update, cfg, db)
			} else {
				sendMessage(bot, update.Message.Chat.ID, "❌ У вас нет доступа к этой функции")
			}
		case "📈 Аналитика":
			if utils.IsAdmin(update.Message.From.ID, cfg) {
				Analytics(bot, update, cfg, db)
			} else {
				sendMessage(bot, update.Message.Chat.ID, "❌ У вас нет доступа к этой функции")
			}
		case "🔧 Меню":
			Menu(bot, update, cfg, db)
		default:
			Help(bot, update)
		}
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}