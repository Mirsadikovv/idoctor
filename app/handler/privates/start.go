package handlers

import (
	"log"
	"strconv"

	"idoctor-bot/app/config"
	"idoctor-bot/app/models"
	"idoctor-bot/app/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func Start(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB) {
	var user models.User
	
	// Проверяем существует ли пользователь
	result := db.Where("telegram_id = ?", update.Message.From.ID).First(&user)
	
	if result.Error == gorm.ErrRecordNotFound {
		// Создаем нового пользователя
		user = models.User{
			TelegramID: update.Message.From.ID,
			Name:       update.Message.From.FirstName,
			Username:   &update.Message.From.UserName,
			FirstName:  &update.Message.From.FirstName,
			LastName:   &update.Message.From.LastName,
			Role:       models.UserRoleMaster, // По умолчанию мастер
			IsActive:   true,
		}

		// Если это админ, устанавливаем роль админа
		if utils.IsAdmin(update.Message.From.ID, cfg) {
			user.Role = models.UserRoleAdmin
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("Ошибка создания пользователя: %v", err)
			sendMessage(bot, update.Message.Chat.ID, "❌ Ошибка регистрации. Попробуйте позже.")
			return
		}

		// Уведомляем админов о новом пользователе
		if user.Role == models.UserRoleMaster {
			notifyAdminsNewUser(bot, cfg, &user)
		}

		sendMessage(bot, update.Message.Chat.ID, "✅ Вы успешно зарегистрированы!")
	}

	// Отправляем приветственное сообщение и меню
	welcomeMessage := "🔧 Добро пожаловать в систему ремонтной мастерской!\n\n"
	
	if user.Role == models.UserRoleAdmin {
		welcomeMessage += "👑 Вы вошли как администратор\n"
		welcomeMessage += "У вас есть полный доступ к системе.\n\n"
	} else {
		welcomeMessage += "🔨 Вы вошли как мастер\n"
		welcomeMessage += "Вы можете просматривать и управлять своими заказами.\n\n"
	}

	welcomeMessage += "Используйте меню ниже для навигации:"

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, welcomeMessage)
	msg.ReplyMarkup = getMainKeyboard(user.Role == models.UserRoleAdmin)

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func notifyAdminsNewUser(bot *tgbotapi.BotAPI, cfg *config.Config, user *models.User) {
	message := "👤 Новый пользователь зарегистрировался:\n\n"
	message += "• Имя: " + user.FullName() + "\n"
	if user.Username != nil && *user.Username != "" {
		message += "• Username: @" + *user.Username + "\n"
	}
	message += "• Telegram ID: " + strconv.FormatInt(user.TelegramID, 10) + "\n"
	message += "• Роль: " + string(user.Role)

	for _, idStr := range cfg.Bot.AdminIds {
		if idStr == "" {
			continue
		}
		
		adminId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}

		msg := tgbotapi.NewMessage(adminId, message)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки уведомления админу: %v", err)
		}
	}
}