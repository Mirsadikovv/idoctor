package keyboard

import "fmt"

const SendContact = `
	{
		"keyboard": [[
			{
				"text": "%s",
				"request_contact": true
			}
		]
		],
		"resize_keyboard": true,
		"one_time_keyboard": true
	}`

var (
	SendContactSend = map[string]string{
		"uz": "Raqamni yuborish 📞",
		"ru": "Поделиться контактом 📞",
		"en": "Share contact 📞",
	}
)

var SendContactMap = map[string]string{
	"uz": fmt.Sprintf(SendContact, SendContactSend["uz"]),
	"ru": fmt.Sprintf(SendContact, SendContactSend["ru"]),
	"en": fmt.Sprintf(SendContact, SendContactSend["en"]),
}
