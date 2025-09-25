package handlers

import (
	"log"

	"idoctor-bot/app/config"
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

func Menu(bot *tgbotapi.BotAPI, update tgbotapi.Update, cfg *config.Config, db *gorm.DB, langCache *i18n.LanguageCache) {
	var user models.User
	if err := db.Where("telegram_id = ?", update.Message.From.ID).First(&user).Error; err != nil {
		sendMessage(bot, update.Message.Chat.ID, i18n.GetText(i18n.UserNotFound, langCache.Get(update.Message.From.ID)))
		return
	}

	lang := user.GetLanguage()
	langCache.Set(update.Message.From.ID, lang)

	welcomeMessage := i18n.GetText(i18n.WelcomeMessage, lang) + "\n\n"

	if user.Role == models.UserRoleAdmin {
		welcomeMessage += i18n.GetText(i18n.AdminWelcome, lang) + "\n\n"
	} else {
		welcomeMessage += i18n.GetText(i18n.MasterWelcome, lang) + "\n\n"
	}

	welcomeMessage += i18n.GetText(i18n.UseMenuBelow, lang)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, welcomeMessage)
	msg.ReplyMarkup = getMainKeyboard(user.Role, lang)

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending menu: %v", err)
	}
}

func Help(bot *tgbotapi.BotAPI, update tgbotapi.Update, langCache *i18n.LanguageCache) {
	lang := langCache.Get(update.Message.From.ID)

	// Создаем inline клавиатуру для дополнительных опций
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "📞 Связаться с поддержкой",
					"uz": "📞 Qo'llab-quvvatlash bilan bog'lanish",
					"en": "📞 Contact Support",
				}, lang),
				"help_contact",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				i18n.GetText(map[string]string{
					"ru": "🔧 О системе",
					"uz": "🔧 Tizim haqida",
					"en": "🔧 About System",
				}, lang),
				"help_about",
			),
		),
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.GetText(i18n.HelpText, lang))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending help: %v", err)
	}
}
