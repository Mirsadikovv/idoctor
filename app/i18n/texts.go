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

var ClientWelcome = map[string]string{
	"ru": "👤 Вы вошли как клиент\nВы можете создавать заказы на ремонт и отслеживать их статус.",
	"uz": "👤 Siz mijoz sifatida kirdingiz\nTa'mirlash buyurtmalarini yaratishingiz va ularning holatini kuzatishingiz mumkin.",
	"en": "👤 You are logged in as client\nYou can create repair orders and track their status.",
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
	"search": {
		"ru": "🔍 Поиск",
		"uz": "🔍 Qidiruv",
		"en": "🔍 Search",
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

// Тексты для клиентов
var CreateOrder = map[string]string{
	"ru": "📝 Создать заказ",
	"uz": "📝 Buyurtma yaratish",
	"en": "📝 Create order",
}

var MyOrders = map[string]string{
	"ru": "📋 Мои заказы",
	"uz": "📋 Mening buyurtmalarim",
	"en": "📋 My orders",
}

var OrderStatus = map[string]string{
	"ru": "📊 Статус заказа",
	"uz": "📊 Buyurtma holati",
	"en": "📊 Order status",
}

var PendingOrders = map[string]string{
	"ru": "⏳ Заказы в ожидании",
	"uz": "⏳ Kutilayotgan buyurtmalar",
	"en": "⏳ Pending orders",
}

var AcceptOrder = map[string]string{
	"ru": "✅ Принять заказ",
	"uz": "✅ Buyurtmani qabul qilish",
	"en": "✅ Accept order",
}

var OrderAccepted = map[string]string{
	"ru": "✅ Заказ принят мастером",
	"uz": "✅ Buyurtma usta tomonidan qabul qilindi",
	"en": "✅ Order accepted by master",
}

var OrderDeclined = map[string]string{
	"ru": "❌ Заказ отклонен",
	"uz": "❌ Buyurtma rad etildi",
	"en": "❌ Order declined",
}

var NoAvailableOrders = map[string]string{
	"ru": "📭 Нет доступных заказов",
	"uz": "📭 Mavjud buyurtmalar yo'q",
	"en": "📭 No available orders",
}

// Тексты для создания заказов клиентами
var ClientOrderStart = map[string]string{
	"ru": "📝 Создание нового заказа\n\nДавайте оформим заказ на ремонт вашего устройства.\nВведите тип устройства (например: смартфон, планшет, ноутбук):",
	"uz": "📝 Yangi buyurtma yaratish\n\nQurilmangizni ta'mirlash uchun buyurtma berish jarayonini boshlaymiz.\nQurilma turini kiriting (masalan: smartfon, planshet, noutbuk):",
	"en": "📝 Creating a new order\n\nLet's create a repair order for your device.\nEnter the device type (e.g.: smartphone, tablet, laptop):",
}

var ClientOrderDeviceBrand = map[string]string{
	"ru": "📱 Отлично! Теперь введите марку устройства (например: Apple, Samsung, Xiaomi):",
	"uz": "📱 Ajoyib! Endi qurilma markasini kiriting (masalan: Apple, Samsung, Xiaomi):",
	"en": "📱 Great! Now enter the device brand (e.g.: Apple, Samsung, Xiaomi):",
}

var ClientOrderDeviceModel = map[string]string{
	"ru": "🏷️ Теперь введите модель устройства (например: iPhone 13, Galaxy S21):",
	"uz": "🏷️ Endi qurilma modelini kiriting (masalan: iPhone 13, Galaxy S21):",
	"en": "🏷️ Now enter the device model (e.g.: iPhone 13, Galaxy S21):",
}

var ClientOrderProblem = map[string]string{
	"ru": "❗ Опишите проблему с устройством подробно:",
	"uz": "❗ Qurilma bilan bog'liq muammoni batafsil tavsiflang:",
	"en": "❗ Describe the device problem in detail:",
}

var ClientOrderContact = map[string]string{
	"ru": "📞 Укажите ваше имя и номер телефона для связи в формате:\nИван Иванов\n+998901234567",
	"uz": "📞 Aloqa uchun ismingiz va telefon raqamingizni quyidagi formatda kiriting:\nIvan Ivanov\n+998901234567",
	"en": "📞 Provide your name and phone number for contact in the format:\nIvan Ivanov\n+998901234567",
}

var ClientOrderConfirm = map[string]string{
	"ru": "✅ Проверьте данные заказа:\n\n📱 Тип устройства: %s\n🏷️ Марка: %s\n📋 Модель: %s\n❗ Проблема: %s\n👤 Контакт: %s\n📞 Телефон: %s\n\nВсе верно?",
	"uz": "✅ Buyurtma ma'lumotlarini tekshiring:\n\n📱 Qurilma turi: %s\n🏷️ Marka: %s\n📋 Model: %s\n❗ Muammo: %s\n👤 Kontakt: %s\n📞 Telefon: %s\n\nHammasi to'g'rimi?",
	"en": "✅ Check your order details:\n\n📱 Device type: %s\n🏷️ Brand: %s\n📋 Model: %s\n❗ Problem: %s\n👤 Contact: %s\n📞 Phone: %s\n\nIs everything correct?",
}

var ClientOrderCreated = map[string]string{
	"ru": "🎉 Заказ успешно создан!\n\n📋 Номер заказа: %s\n\nВаш заказ передан мастерам. Как только кто-то из них примет заказ, мы уведомим вас.",
	"uz": "🎉 Buyurtma muvaffaqiyatli yaratildi!\n\n📋 Buyurtma raqami: %s\n\nBuyurtmangiz ustalarga yuborildi. Ulardan biri buyurtmani qabul qilganda sizni xabardor qilamiz.",
	"en": "🎉 Order created successfully!\n\n📋 Order number: %s\n\nYour order has been sent to masters. We'll notify you once one of them accepts the order.",
}

var ClientOrderCancelled = map[string]string{
	"ru": "❌ Создание заказа отменено",
	"uz": "❌ Buyurtma yaratish bekor qilindi",
	"en": "❌ Order creation cancelled",
}

var ConfirmYes = map[string]string{
	"ru": "✅ Да, все верно",
	"uz": "✅ Ha, hammasi to'g'ri",
	"en": "✅ Yes, everything is correct",
}

var ConfirmNo = map[string]string{
	"ru": "❌ Нет, изменить",
	"uz": "❌ Yo'q, o'zgartirish",
	"en": "❌ No, change",
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
