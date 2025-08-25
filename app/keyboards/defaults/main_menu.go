package keyboard

import "fmt"

const MainMenuKeyboard = `
	{
		"keyboard": [[
			{
				"text": "%s",
				"callback_data": "profile"
			},
			{
				"text": "%s",
				"callback_data": "about"
			}
		],
		[
			{
				"text": "%s",
				"web_app": {
					"url": "https://dmi.inspector.mehnat.sriss.uz/ru/web-form"
				}
			}
		],
		[
			{
				"text": "%s",
				"callback_data": "language"
			}
		]
		],
		"resize_keyboard": true,
		"one_time_keyboard": true
	}`

var (
	AppealKeyboardProfile = map[string]string{
		"uz": "Profil👤",
		"ru": "Профиль👤",
		"en": "Profile👤",
	}
	AppealKeyboardAbout = map[string]string{
		"ru": "О насℹ️",
		"uz": "Biz haqimizdaℹ️",
		"en": "About usℹ️",
	}
	AppealKeyboardSend = map[string]string{
		"ru": "Подать заявку📄",
		"uz": "Murojaat yo'llash📄",
		"en": "Send an application📄",
	}

	AppealKeyboardLanguage = map[string]string{
		"ru": "Поменять язык🌐",
		"uz": "Tilni o'zgartirish🌐",
		"en": "Change language🌐",
	}
)

var MainMenuKeyboardMap = map[string]string{
	"uz": fmt.Sprintf(MainMenuKeyboard, AppealKeyboardProfile["uz"], AppealKeyboardAbout["uz"], AppealKeyboardSend["uz"], AppealKeyboardLanguage["uz"]),
	"ru": fmt.Sprintf(MainMenuKeyboard, AppealKeyboardProfile["ru"], AppealKeyboardAbout["ru"], AppealKeyboardSend["ru"], AppealKeyboardLanguage["ru"]),
	"en": fmt.Sprintf(MainMenuKeyboard, AppealKeyboardProfile["en"], AppealKeyboardAbout["en"], AppealKeyboardSend["en"], AppealKeyboardLanguage["en"]),
}

var Back = map[string]string{
	"ru": "Назад ⬅️",
	"uz": "Ortga ⬅️",
	"en": "Back ⬅️",
}
