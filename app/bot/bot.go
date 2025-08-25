package bot

import (
	"log"

	"idoctor-bot/app/config"
	handlers "idoctor-bot/app/handler/privates"
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
	)
	if err != nil {
		log.Printf("Ошибка автомиграции: %v", err)
		return err
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
			handlers.HandleUpdate(bot, update, cfg, db)
		}
	}

	return nil
}
