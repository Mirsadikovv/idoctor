package handlers

import (
	"fmt"
	
	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"
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
				tgbotapi.NewKeyboardButton(i18n.GetButton("new_order", lang)),
			},
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("search", lang)),
				tgbotapi.NewKeyboardButton(i18n.GetButton("menu", lang)),
			},
			{
				tgbotapi.NewKeyboardButton(i18n.GetButton("change_language", lang)),
			},
		}
	}

	return tgbotapi.NewReplyKeyboard(buttons...)
}

func getMainInlineKeyboard(isAdmin bool, lang string) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton

	if isAdmin {
		// Клавиатура для админа
		buttons = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("all_orders", lang), "all_orders"),
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("new_order", lang), "new_order"),
			},
			{
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("masters", lang), "main_masters"),
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("analytics", lang), "analytics"),
			},
			{
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("search", lang), "search"),
			},
		}
	} else {
		// Клавиатура для мастера
		buttons = [][]tgbotapi.InlineKeyboardButton{
			{
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("my_orders", lang), "my_orders"),
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("new_order", lang), "new_order"),
			},
			{
				tgbotapi.NewInlineKeyboardButtonData(i18n.GetButton("search", lang), "search"),
			},
		}
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
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
		tgbotapi.NewInlineKeyboardButtonData("🔄 Изменить статус", fmt.Sprintf("status_change_%d", orderID)),
		tgbotapi.NewInlineKeyboardButtonData("💰 Установить цену", fmt.Sprintf("price_set_%d", orderID)),
	})

	if isAdmin {
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👨‍🔧 Назначить мастера", fmt.Sprintf("master_assign_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("order_edit_%d", orderID)),
		})
	}

	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("ℹ️ Подробно", fmt.Sprintf("order_details_%d", orderID)),
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

// getStatisticsMainKeyboard возвращает главное меню статистики
func getStatisticsMainKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📊 Общая статистика",
					"uz": "📊 Umumiy statistika",
					"en": "📊 General statistics",
				}, lang),
				"stats_general"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "👥 По мастерам",
					"uz": "👥 Ustalar bo'yicha",
					"en": "👥 By masters",
				}, lang),
				"stats_masters"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📅 Сегодня",
					"uz": "📅 Bugun",
					"en": "📅 Today",
				}, lang),
				"stats_period_today"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📅 Вчера",
					"uz": "📅 Kecha",
					"en": "📅 Yesterday",
				}, lang),
				"stats_period_yesterday"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📅 Эта неделя",
					"uz": "📅 Bu hafta",
					"en": "📅 This week",
				}, lang),
				"stats_period_week"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📅 Этот месяц",
					"uz": "📅 Bu oy",
					"en": "📅 This month",
				}, lang),
				"stats_period_month"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📅 Этот год",
					"uz": "📅 Bu yil",
					"en": "📅 This year",
				}, lang),
				"stats_period_year"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📊 За все время",
					"uz": "📊 Barcha vaqt",
					"en": "📊 All time",
				}, lang),
				"stats_period_all_time"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "🔄 Обновить",
					"uz": "🔄 Yangilash",
					"en": "🔄 Refresh",
				}, lang),
				"stats_refresh"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "🏠 Главное меню",
					"uz": "🏠 Asosiy menyu",
					"en": "🏠 Main menu",
				}, lang),
				"main_menu"),
		},
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// getStatisticsBackKeyboard возвращает кнопку возврата к статистике
func getStatisticsBackKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📊 К статистике",
					"uz": "📊 Statistikaga",
					"en": "📊 To statistics",
				}, lang),
				"back_to_stats"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "🏠 Главное меню",
					"uz": "🏠 Asosiy menyu",
					"en": "🏠 Main menu",
				}, lang),
				"main_menu"),
		},
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// getMastersMainKeyboard возвращает главное меню управления мастерами
func getMastersMainKeyboard(lang string) tgbotapi.InlineKeyboardMarkup {
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "👥 Все мастера",
					"uz": "👥 Barcha ustalar",
					"en": "👥 All masters",
				}, lang),
				"masters_list_all"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "✅ Активные",
					"uz": "✅ Faol",
					"en": "✅ Active",
				}, lang),
				"masters_list_active"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "➕ Добавить мастера",
					"uz": "➕ Usta qo'shish",
					"en": "➕ Add master",
				}, lang),
				"masters_add"),
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "📊 Статистика",
					"uz": "📊 Statistika",
					"en": "📊 Statistics",
				}, lang),
				"stats_masters"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "🔄 Обновить",
					"uz": "🔄 Yangilash",
					"en": "🔄 Refresh",
				}, lang),
				"masters_refresh"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData(
				getText(map[string]string{
					"ru": "🏠 Главное меню",
					"uz": "🏠 Asosiy menyu",
					"en": "🏠 Main menu",
				}, lang),
				"main_menu"),
		},
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// getMasterListKeyboard возвращает клавиатуру для списка мастеров
func getMasterListKeyboard(masters []models.User, lang string) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// Добавляем кнопки для каждого мастера (максимум 8 для читаемости)
	maxMasters := len(masters)
	if maxMasters > 8 {
		maxMasters = 8
	}
	
	for i := 0; i < maxMasters; i += 2 {
		var row []tgbotapi.InlineKeyboardButton
		
		// Первый мастер в ряду
		master := masters[i]
		statusIcon := "✅"
		if !master.IsActive {
			statusIcon = "❌"
		}
		
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", statusIcon, master.FullName()),
			fmt.Sprintf("master_action_profile_%d", master.ID)))
		
		// Второй мастер в ряду (если есть)
		if i+1 < maxMasters {
			master2 := masters[i+1]
			statusIcon2 := "✅"
			if !master2.IsActive {
				statusIcon2 = "❌"
			}
			
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %s", statusIcon2, master2.FullName()),
				fmt.Sprintf("master_action_profile_%d", master2.ID)))
		}
		
		buttons = append(buttons, row)
	}
	
	// Кнопки навигации и управления
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "🔄 Обновить",
				"uz": "🔄 Yangilash",
				"en": "🔄 Refresh",
			}, lang),
			"masters_refresh"),
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "📊 Статистика",
				"uz": "📊 Statistika",
				"en": "📊 Statistics",
			}, lang),
			"stats_masters"),
	})
	
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "🔙 К мастерам",
				"uz": "🔙 Ustalarga",
				"en": "🔙 Back to masters",
			}, lang),
			"back_to_masters"),
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "🏠 Главное меню",
				"uz": "🏠 Asosiy menyu",
				"en": "🏠 Main menu",
			}, lang),
			"main_menu"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// getMasterProfileKeyboard возвращает клавиатуру для профиля мастера
func getMasterProfileKeyboard(masterID uint, isActive bool, lang string) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// Кнопка изменения статуса
	statusText := getText(map[string]string{
		"ru": "❌ Деактивировать",
		"uz": "❌ Faolsizlantirish",
		"en": "❌ Deactivate",
	}, lang)
	
	if !isActive {
		statusText = getText(map[string]string{
			"ru": "✅ Активировать",
			"uz": "✅ Faollashtirish",
			"en": "✅ Activate",
		}, lang)
	}
	
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(statusText, fmt.Sprintf("master_action_toggle_%d", masterID)),
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "📋 Заказы мастера",
				"uz": "📋 Usta buyurtmalari",
				"en": "📋 Master orders",
			}, lang),
			fmt.Sprintf("master_action_orders_%d", masterID)),
	})
	
	// Кнопки навигации
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "🔄 Обновить",
				"uz": "🔄 Yangilash",
				"en": "🔄 Refresh",
			}, lang),
			fmt.Sprintf("master_action_refresh_%d", masterID)),
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "👥 К списку",
				"uz": "👥 Ro'yxatga",
				"en": "👥 To list",
			}, lang),
			"masters_list_all"),
	})
	
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "🔙 К мастерам",
				"uz": "🔙 Ustalarga",
				"en": "🔙 Back to masters",
			}, lang),
			"back_to_masters"),
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "🏠 Главное меню",
				"uz": "🏠 Asosiy menyu",
				"en": "🏠 Main menu",
			}, lang),
			"main_menu"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// getMasterAssignKeyboard возвращает клавиатуру для назначения мастера на заказ
func getMasterAssignKeyboard(deviceID uint, masters []models.User, lang string) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton
	
	// Добавляем кнопки для каждого активного мастера
	for _, master := range masters {
		if !master.IsActive {
			continue // Пропускаем неактивных мастеров
		}
		
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("👨‍🔧 %s", master.FullName()),
				fmt.Sprintf("assign_master_%d_%d", deviceID, master.ID)),
		})
	}
	
	// Кнопка отмены
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			getText(map[string]string{
				"ru": "❌ Отмена",
				"uz": "❌ Bekor qilish",
				"en": "❌ Cancel",
			}, lang),
			"back_to_orders"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// getText - вспомогательная функция для получения локализованного текста
func getText(texts map[string]string, lang string) string {
	if text, exists := texts[lang]; exists {
		return text
	}
	return texts["en"] // Fallback to English
}