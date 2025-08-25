package handlers

import (
	"log"

	"idoctor-bot/app/config"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func ShowLanguageMenu(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	var user models.User
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.UserNotFound, langCache.Get(update.Message.From.ID)))
		return
	}

	lang := user.GetLanguage()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.ChooseLanguage, lang))
	msg.ReplyMarkup = getLanguageKeyboard()

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending language menu: %v", err)
	}
}

func ChangeLanguage(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	var newLang string

	switch update.Message.Text {
	case "🇷🇺 Русский":
		newLang = "ru"
	case "🇺🇿 O'zbek":
		newLang = "uz"
	case "🇬🇧 English":
		newLang = "en"
	default:
		ShowLanguageMenu(bot, update, cfg, db, langCache)
		return
	}

	// Обновляем язык в базе данных
	if err := db.Model(&models.User{}).
		Where("telegram_id = ?", update.Message.From.ID).
		Update("language", newLang).Error; err != nil {
		log.Printf("Error updating user language: %v", err)
		sendMessage(bot, update.Message.Chat.ID, "❌ Ошибка обновления языка")
		return
	}

	// Обновляем кэш
	langCache.Set(update.Message.From.ID, newLang)

	// Отправляем подтверждение на новом языке
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.LanguageChanged, newLang))
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending language confirmation: %v", err)
	}

	// Показываем главное меню на новом языке
	Menu(bot, update, cfg, db, langCache)
}

func getLanguageKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🇷🇺 Русский"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🇺🇿 O'zbek"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🇬🇧 English"),
		),
	)
}