package handlers

import (
	"log"

	"idoctor-bot/app/config"
	"idoctor-bot/app/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func Menu(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	var user models.User
	
	// Получаем пользователя из базы данных
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		sendMessage(bot, update.Message.Chat.ID, "❌ Пользователь не найден. Используйте /start для регистрации.")
		return
	}

	menuMessage := "🔧 Главное меню\n\n"
	
	if user.Role == models.UserRoleAdmin {
		menuMessage += "👑 Панель администратора:\n\n"
		menuMessage += "📊 Все заказы - Просмотр всех заказов в системе\n"
		menuMessage += "➕ Новый заказ - Создать новый заказ\n"
		menuMessage += "👥 Мастера - Управление мастерами\n"
		menuMessage += "📈 Аналитика - Статистика и отчеты\n"
	} else {
		menuMessage += "🔨 Панель мастера:\n\n"
		menuMessage += "📋 Мои заказы - Просмотр ваших заказов\n"
		menuMessage += "🔧 Меню - Это меню\n"
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, menuMessage)
	msg.ReplyMarkup = getMainKeyboard(user.Role == models.UserRoleAdmin)

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки меню: %v", err)
	}
}

func Help(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	helpMessage := "🆘 Помощь - Система ремонтной мастерской\n\n"
	helpMessage += "📱 Доступные команды:\n"
	helpMessage += "• /start - Регистрация в системе\n"
	helpMessage += "• /menu - Главное меню\n"
	helpMessage += "• /orders - Просмотр заказов\n"
	helpMessage += "• /help - Эта справка\n\n"
	helpMessage += "🔧 Функции системы:\n"
	helpMessage += "• Управление заказами\n"
	helpMessage += "• Отслеживание статуса ремонта\n"
	helpMessage += "• Управление запчастями\n"
	helpMessage += "• Аналитика и отчеты (для админов)\n\n"
	helpMessage += "❓ По вопросам обращайтесь к администратору."

	sendMessage(bot, update.Message.Chat.ID, helpMessage)
}