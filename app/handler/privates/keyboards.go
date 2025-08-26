package handlers

import (
	"idoctor-bot/app/i18n"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getMainKeyboard(isAdmin bool, lang string) tgbotapi.ReplyKeyboardMarkup {
	var buttons [][]tgbotapi.KeyboardButton

	if isAdmin {
		// Клавиатура для админа
		buttons = [][]tgbotapi.KeyboardButton{
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("all_orders", lang)),
				tgbotapi.NewKeyboardButton(i18n.GetButton("new_order", lang)),
			},
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("masters", lang)),
				tgbotapi.NewKeyboardButton(i18n.GetButton("analytics", lang)),
			},
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("search", lang)),
				tgbotapi.NewKeyboardButton(i18n.GetButton("menu", lang)),
			},
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("change_language", lang)),
			},
		}
	} else {
		// Клавиатура для мастера
		buttons = [][]tgbotapi.KeyboardButton{
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("my_orders", lang)),
				tgbotapi.NewKeyboardButton(i18n.GetButton("search", lang)),
			},
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("menu", lang)),
			},
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("change_language", lang)),
			},
		}
	}

	return tgbotapi.NewReplyKeyboard(buttons...)
}

func getDeviceStatusKeyboard() tgbotapi.InlineKeyboardMarkup {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🆕 Принят", "status_received"),
			tgbotapi.NewInlineKeyboardButtonData("🔧 В работе", "status_in_progress"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⏳ Ожидание запчастей", "status_waiting_parts"),
			tgbotapi.NewInlineKeyboardButtonData("✅ Готов", "status_ready"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📦 Выдан", "status_completed"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменен", "status_cancelled"),
		},
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

func getOrderActionsKeyboard(orderID uint, isAdmin bool) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton

	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 Изменить статус", "change_status_"+string(rune(orderID))),
		tgbotapi.NewInlineKeyboardButtonData("💰 Установить цену", "set_price_"+string(rune(orderID))),
	})

	if isAdmin {
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👨‍🔧 Назначить мастера", "assign_master_"+string(rune(orderID))),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", "edit_order_"+string(rune(orderID))),
		})
	}

	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("ℹ️ Подробно", "details_"+string(rune(orderID))),
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "back_to_orders"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

func getBackKeyboard() tgbotapi.InlineKeyboardMarkup {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к заказам", "back_to_orders"),
		},
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}