package bot

import (
	"log"

	"idoctor-bot/app/config"
	handlers "idoctor-bot/app/handler/privates"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
	"idoctor-bot/app/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func Start(cfg *config.Config, db *gorm.DB) error {
	// Автомиграция моделей
	err := db.AutoMigrate(
		&models.User{},
		&models.Customer{},
		&models.Device{},
		&models.Part{},
		&models.UserState{},
	)
	if err != nil {
		log.Printf("Ошибка автомиграции: %v", err)
		return err
	}

	// Инициализация кэша языков
	langCache := i18n.NewLanguageCache()
	
	// Загружаем языки пользователей из базы данных
	var users []models.User
	if err := db.Find(&users).Error; err == nil {
		for _, user := range users {
			langCache.Set(user.TelegramID, user.GetLanguage())
		}
		log.Printf("Загружено языков пользователей: %d", len(users))
	}

	bot, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		return err
	}

	bot.Debug = cfg.Bot.Debug

	log.Printf("Бот запущен: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = cfg.Bot.Timeout

	handlers.Register(bot, db)

	utils.NotifyAdmins(bot, cfg)

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handlers.HandleUpdate(bot, update, cfg, db, langCache)
		} else if update.CallbackQuery != nil {
			handlers.HandleCallback(bot, update.CallbackQuery, cfg, db, langCache)
		}
	}

	return nil
}
