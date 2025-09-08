package handlers

import (
	"log"
	"strconv"

	"idoctor-bot/app/config"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
	"idoctor-bot/app/services"
	"idoctor-bot/app/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func Start(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	// Очищаем состояние пользователя при команде /start
	stateService := services.NewStateService(db)
	stateService.ClearState(update.Message.From.ID)
	
	var user models.User
	
	// Проверяем существует ли пользователь
	result := db.Where("telegram_id = ?", update.Message.From.ID).First(&user)
	
	if result.Error == gorm.ErrRecordNotFound {
		// Определяем язык по умолчанию
		lang := "ru"
		if update.Message.From.LanguageCode != "" {
			switch update.Message.From.LanguageCode {
			case "uz":
				lang = "uz"
			case "en":
				lang = "en"
			}
		}

		// Создаем нового пользователя
		user = models.User{
			TelegramID: update.Message.From.ID,
			Name:       update.Message.From.FirstName,
			Username:   &update.Message.From.UserName,
			FirstName:  &update.Message.From.FirstName,
			LastName:   &update.Message.From.LastName,
			Role:       models.UserRoleMaster, // По умолчанию мастер
			Language:   lang,
			IsActive:   true,
		}

		// Если это админ, устанавливаем роль админа
		if utils.IsAdmin(update.Message.From.ID, cfg) {
			user.Role = models.UserRoleAdmin
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("Ошибка создания пользователя: %v", err)
			sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.RegistrationError, lang))
			return
		}

		// Уведомляем админов о новом пользователе
		if user.Role == models.UserRoleMaster {
			notifyAdminsNewUser(bot, cfg, &user)
		}

		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.RegistrationSuccess, user.GetLanguage()))
	}

	// Обновляем кэш языка
	langCache.Set(update.Message.From.ID, user.GetLanguage())

	// Отправляем приветственное сообщение и меню
	lang := user.GetLanguage()
	welcomeMessage := i18n.GetText(i18n.WelcomeMessage, lang) + "\n\n"
	
	if user.Role == models.UserRoleAdmin {
		welcomeMessage += i18n.GetText(i18n.AdminWelcome, lang) + "\n\n"
	} else {
		welcomeMessage += i18n.GetText(i18n.MasterWelcome, lang) + "\n\n"
	}

	welcomeMessage += i18n.GetText(i18n.UseMenuBelow, lang)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, welcomeMessage)
	msg.ReplyMarkup = getMainKeyboard(user.Role == models.UserRoleAdmin, lang)

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func notifyAdminsNewUser(bot *tgbotapi.BotAPI, cfg *config.Config, user *models.User) {
	// Для уведомлений админов используем русский язык
	message := i18n.GetText(i18n.NewUserRegistered, "ru") + "\n\n"
	message += i18n.GetText(i18n.Name, "ru") + " " + user.FullName() + "\n"
	if user.Username != nil && *user.Username != "" {
		message += i18n.GetText(i18n.Username, "ru") + " @" + *user.Username + "\n"
	}
	message += i18n.GetText(i18n.TelegramID, "ru") + " " + strconv.FormatInt(user.TelegramID, 10) + "\n"
	message += i18n.GetText(i18n.Role, "ru") + " " + string(user.Role)

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