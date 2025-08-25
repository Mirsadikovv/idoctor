package utils

import (
	"log"
	"strconv"

	"idoctor-bot/app/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func NotifyAdmins(bot *tgbotapi.BotAPI, cfg *config.Config) {
	message := "🔧 Бот ремонтной мастерской запущен!\n\n"
	message += "📱 Доступные команды:\n"
	message += "• /start - Регистрация\n"
	message += "• /menu - Главное меню\n"
	message += "• /orders - Заказы\n"
	message += "• /help - Помощь\n\n"
	message += "⚡ Система готова к работе!"

	for _, idStr := range cfg.Bot.AdminIds {
		if idStr == "" {
			continue
		}
		
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("Invalid admin ID: %v", err)
			continue
		}

		msg := tgbotapi.NewMessage(id, message)
		msg.ParseMode = "HTML"

		_, err = bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send admin notification: %v", err)
		}
	}
}

func IsAdmin(userID int64, cfg *config.Config) bool {
	for _, idStr := range cfg.Bot.AdminIds {
		if idStr == "" {
			continue
		}
		
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		
		if id == userID {
			return true
		}
	}
	return false
}
