package utils

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetBotCommands() []tgbotapi.BotCommand {
	return []tgbotapi.BotCommand{
		{Command: "/start", Description: "Регистрация и главное меню"},
		{Command: "/menu", Description: "Главное меню"},
		{Command: "/orders", Description: "Мои заказы"},
		{Command: "/help", Description: "Помощь"},
	}
}
