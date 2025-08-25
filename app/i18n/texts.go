package i18n

// Основные тексты системы
var WelcomeMessage = map[string]string{
	"ru": "🔧 Добро пожаловать в систему ремонтной мастерской!",
	"uz": "🔧 Ta'mirlash ustaxonasi tizimiga xush kelibsiz!",
	"en": "🔧 Welcome to the repair workshop system!",
}

var AdminWelcome = map[string]string{
	"ru": "👑 Вы вошли как администратор\nУ вас есть полный доступ к системе.",
	"uz": "👑 Siz administrator sifatida kirdingiz\nSizda tizimga to'liq ruxsat bor.",
	"en": "👑 You are logged in as administrator\nYou have full access to the system.",
}

var MasterWelcome = map[string]string{
	"ru": "🔨 Вы вошли как мастер\nВы можете просматривать и управлять своими заказами.",
	"uz": "🔨 Siz usta sifatida kirdingiz\nO'z buyurtmalaringizni ko'rishingiz va boshqarishingiz mumkin.",
	"en": "🔨 You are logged in as master\nYou can view and manage your orders.",
}

var UseMenuBelow = map[string]string{
	"ru": "Используйте меню ниже для навигации:",
	"uz": "Navigatsiya uchun quyidagi menyudan foydalaning:",
	"en": "Use the menu below for navigation:",
}

var RegistrationSuccess = map[string]string{
	"ru": "✅ Вы успешно зарегистрированы!",
	"uz": "✅ Siz muvaffaqiyatli ro'yxatdan o'tdingiz!",
	"en": "✅ You have been successfully registered!",
}

var RegistrationError = map[string]string{
	"ru": "❌ Ошибка регистрации. Попробуйте позже.",
	"uz": "❌ Ro'yxatdan o'tishda xatolik. Keyinroq urinib ko'ring.",
	"en": "❌ Registration error. Please try again later.",
}

var NewUserRegistered = map[string]string{
	"ru": "👤 Новый пользователь зарегистрировался:",
	"uz": "👤 Yangi foydalanuvchi ro'yxatdan o'tdi:",
	"en": "👤 New user has registered:",
}

var Name = map[string]string{
	"ru": "• Имя:",
	"uz": "• Ism:",
	"en": "• Name:",
}

var Username = map[string]string{
	"ru": "• Username:",
	"uz": "• Username:",
	"en": "• Username:",
}

var TelegramID = map[string]string{
	"ru": "• Telegram ID:",
	"uz": "• Telegram ID:",
	"en": "• Telegram ID:",
}

var Role = map[string]string{
	"ru": "• Роль:",
	"uz": "• Rol:",
	"en": "• Role:",
}

var NoAccess = map[string]string{
	"ru": "❌ У вас нет доступа к этой функции",
	"uz": "❌ Sizda bu funksiyaga ruxsat yo'q",
	"en": "❌ You don't have access to this function",
}

var UserNotFound = map[string]string{
	"ru": "❌ Пользователь не найден",
	"uz": "❌ Foydalanuvchi topilmadi",
	"en": "❌ User not found",
}

var InvalidDataFormat = map[string]string{
	"ru": "❌ Неверный формат данных",
	"uz": "❌ Noto'g'ri ma'lumot formati",
	"en": "❌ Invalid data format",
}

var InvalidOrderID = map[string]string{
	"ru": "❌ Неверный ID заказа",
	"uz": "❌ Noto'g'ri buyurtma ID",
	"en": "❌ Invalid order ID",
}

var UnknownAction = map[string]string{
	"ru": "❌ Неизвестное действие",
	"uz": "❌ Noma'lum harakat",
	"en": "❌ Unknown action",
}

var HelpText = map[string]string{
	"ru": "ℹ️ *Справка*\n\n" +
		"*Доступные команды:*\n" +
		"/start - Запустить бота\n" +
		"/menu - Главное меню\n" +
		"/orders - Мои заказы\n" +
		"/help - Помощь\n" +
		"/lang - Сменить язык\n\n" +
		"*Кнопки меню:*\n" +
		"📋 Мои заказы - Просмотр ваших заказов\n" +
		"📊 Все заказы - Все заказы (только для админов)\n" +
		"➕ Новый заказ - Создать новый заказ (только для админов)\n" +
		"👥 Мастера - Управление мастерами (только для админов)\n" +
		"📈 Аналитика - Статистика (только для админов)",

	"uz": "ℹ️ *Yordam*\n\n" +
		"*Mavjud buyruqlar:*\n" +
		"/start - Botni ishga tushirish\n" +
		"/menu - Asosiy menyu\n" +
		"/orders - Mening buyurtmalarim\n" +
		"/help - Yordam\n" +
		"/lang - Tilni o'zgartirish\n\n" +
		"*Menyu tugmalari:*\n" +
		"📋 Mening buyurtmalarim - Buyurtmalaringizni ko'rish\n" +
		"📊 Barcha buyurtmalar - Barcha buyurtmalar (faqat adminlar uchun)\n" +
		"➕ Yangi buyurtma - Yangi buyurtma yaratish (faqat adminlar uchun)\n" +
		"👥 Ustalar - Ustalarni boshqarish (faqat adminlar uchun)\n" +
		"📈 Analitika - Statistika (faqat adminlar uchun)",

	"en": "ℹ️ *Help*\n\n" +
		"*Available commands:*\n" +
		"/start - Start the bot\n" +
		"/menu - Main menu\n" +
		"/orders - My orders\n" +
		"/help - Help\n" +
		"/lang - Change language\n\n" +
		"*Menu buttons:*\n" +
		"📋 My orders - View your orders\n" +
		"📊 All orders - All orders (admin only)\n" +
		"➕ New order - Create new order (admin only)\n" +
		"👥 Masters - Manage masters (admin only)\n" +
		"📈 Analytics - Statistics (admin only)",
}

var ChooseLanguage = map[string]string{
	"ru": "🌐 Выберите язык:",
	"uz": "🌐 Tilni tanlang:",
	"en": "🌐 Choose language:",
}

var LanguageChanged = map[string]string{
	"ru": "✅ Язык изменен на русский",
	"uz": "✅ Til o'zbek tiliga o'zgartirildi",
	"en": "✅ Language changed to English",
}

// Кнопки клавиатуры
var Buttons = map[string]map[string]string{
	"my_orders": {
		"ru": "📋 Мои заказы",
		"uz": "📋 Mening buyurtmalarim",
		"en": "📋 My orders",
	},
	"all_orders": {
		"ru": "📊 Все заказы",
		"uz": "📊 Barcha buyurtmalar",
		"en": "📊 All orders",
	},
	"new_order": {
		"ru": "➕ Новый заказ",
		"uz": "➕ Yangi buyurtma",
		"en": "➕ New order",
	},
	"masters": {
		"ru": "👥 Мастера",
		"uz": "👥 Ustalar",
		"en": "👥 Masters",
	},
	"analytics": {
		"ru": "📈 Аналитика",
		"uz": "📈 Analitika",
		"en": "📈 Analytics",
	},
	"menu": {
		"ru": "🔧 Меню",
		"uz": "🔧 Menyu",
		"en": "🔧 Menu",
	},
	"change_language": {
		"ru": "🌐 Изменить язык",
		"uz": "🌐 Tilni o'zgartirish",
		"en": "🌐 Change language",
	},
	"russian": {
		"ru": "🇷🇺 Русский",
		"uz": "🇷🇺 Русский",
		"en": "🇷🇺 Русский",
	},
	"uzbek": {
		"ru": "🇺🇿 O'zbek",
		"uz": "🇺🇿 O'zbek",
		"en": "🇺🇿 O'zbek",
	},
	"english": {
		"ru": "🇬🇧 English",
		"uz": "🇬🇧 English",
		"en": "🇬🇧 English",
	},
}

// Функция для получения текста по языку
func GetText(textMap map[string]string, lang string) string {
	if text, exists := textMap[lang]; exists {
		return text
	}
	return textMap["ru"] // fallback на русский
}

// Функция для получения текста кнопки
func GetButton(key, lang string) string {
	if buttonMap, exists := Buttons[key]; exists {
		return GetText(buttonMap, lang)
	}
	return key // fallback на ключ
}