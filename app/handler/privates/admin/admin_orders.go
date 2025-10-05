package admin

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"idoctor-bot/app/i18n"
	"idoctor-bot/app/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// HandleAdminOrders показывает все заказы для администратора
func HandleAdminOrders(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	_ = langCache.Get(callbackQuery.From.ID) // lang не используется пока

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для просмотра всех заказов")
		return
	}

	// Получаем все заказы
	var devices []models.Device
	if err := db.Preload("Customer").Preload("Master").Order("created_at DESC").Find(&devices).Error; err != nil {
		log.Printf("Error getting devices: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения заказов")
		return
	}

	if len(devices) == 0 {
		msg := tgbotapi.NewMessage(callbackQuery.From.ID, "📋 Нет заказов в системе")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "main_menu"),
			),
		)
		bot.Send(msg)
		return
	}

	// Формируем статистику
	statusCount := make(map[models.DeviceStatus]int)
	for _, device := range devices {
		statusCount[device.Status]++
	}

	text := "📋 *Все заказы в системе*\n\n"
	text += fmt.Sprintf("📊 *Всего заказов:* %d\n\n", len(devices))

	// Статистика по статусам
	if count, ok := statusCount[models.DeviceStatusReceived]; ok {
		text += fmt.Sprintf("🆕 Принято: %d\n", count)
	}
	if count, ok := statusCount[models.DeviceStatusInProgress]; ok {
		text += fmt.Sprintf("🔧 В работе: %d\n", count)
	}
	if count, ok := statusCount[models.DeviceStatusWaitingParts]; ok {
		text += fmt.Sprintf("⏳ Ожидание запчастей: %d\n", count)
	}
	if count, ok := statusCount[models.DeviceStatusReady]; ok {
		text += fmt.Sprintf("✅ Готово: %d\n", count)
	}
	if count, ok := statusCount[models.DeviceStatusCompleted]; ok {
		text += fmt.Sprintf("📦 Выдано: %d\n", count)
	}
	if count, ok := statusCount[models.DeviceStatusCancelled]; ok {
		text += fmt.Sprintf("❌ Отменено: %d\n", count)
	}

	text += "\n_Выберите статус для просмотра заказов:_"

	// Создаем клавиатуру с фильтрами по статусам
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕 Принято", "admin_orders_filter_received"),
			tgbotapi.NewInlineKeyboardButtonData("🔧 В работе", "admin_orders_filter_inProgress"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏳ Ожидание", "admin_orders_filter_waitingParts"),
			tgbotapi.NewInlineKeyboardButtonData("✅ Готово", "admin_orders_filter_ready"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Выдано", "admin_orders_filter_completed"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменено", "admin_orders_filter_cancelled"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Все заказы", "admin_orders_filter_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

// HandleAdminOrdersFilter обрабатывает фильтрацию заказов по статусу
func HandleAdminOrdersFilter(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) < 4 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	filterStatus := parts[3]
	lang := langCache.Get(callbackQuery.From.ID)

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для просмотра заказов")
		return
	}

	// Получаем заказы с фильтром
	var devices []models.Device
	query := db.Preload("Customer").Preload("Master").Order("created_at DESC")

	if filterStatus != "all" {
		query = query.Where("status = ?", filterStatus)
	}

	if err := query.Find(&devices).Error; err != nil {
		log.Printf("Error getting devices: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения заказов")
		return
	}

	if len(devices) == 0 {
		statusText := getStatusTextRu(filterStatus)
		msg := tgbotapi.NewMessage(callbackQuery.From.ID, fmt.Sprintf("📋 Нет заказов со статусом: %s", statusText))
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "admin_orders"),
			),
		)
		bot.Send(msg)
		return
	}

	// Показываем заказы (пагинация по 5 штук)
	showAdminOrdersList(bot, callbackQuery.From.ID, devices, filterStatus, 0, lang)
}

// showAdminOrdersList показывает список заказов с пагинацией
func showAdminOrdersList(bot *tgbotapi.BotAPI, chatID int64, devices []models.Device, filterStatus string, page int, lang string) {
	pageSize := 5
	start := page * pageSize
	end := start + pageSize
	if end > len(devices) {
		end = len(devices)
	}

	if start >= len(devices) {
		return
	}

	pageDevices := devices[start:end]
	statusText := getStatusTextRu(filterStatus)

	text := fmt.Sprintf("📋 *Заказы - %s*\n", statusText)
	text += fmt.Sprintf("_Страница %d из %d_\n\n", page+1, (len(devices)+pageSize-1)/pageSize)

	// Создаем клавиатуру с заказами
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, device := range pageDevices {
		deviceText := fmt.Sprintf("🆔 %s - %s %s", device.Code, device.Brand, device.Model)
		if device.Customer != nil {
			deviceText += fmt.Sprintf(" (%s)", device.Customer.Name)
		}

		// Кнопка с заказом
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(deviceText, fmt.Sprintf("admin_order_details_%d", device.ID)),
		})
	}

	// Кнопки пагинации
	var paginationRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		paginationRow = append(paginationRow,
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", fmt.Sprintf("admin_orders_page_%s_%d", filterStatus, page-1)))
	}
	if end < len(devices) {
		paginationRow = append(paginationRow,
			tgbotapi.NewInlineKeyboardButtonData("Далее ▶️", fmt.Sprintf("admin_orders_page_%s_%d", filterStatus, page+1)))
	}
	if len(paginationRow) > 0 {
		keyboard = append(keyboard, paginationRow)
	}

	// Кнопка возврата
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", fmt.Sprintf("admin_orders_filter_%s", filterStatus)),
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "admin_orders"),
	})

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

// HandleAdminOrderDetails показывает детали заказа для администратора
func HandleAdminOrderDetails(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 4 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID заказа")
		return
	}

	_ = langCache.Get(callbackQuery.From.ID) // lang не используется пока

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для просмотра заказа")
		return
	}

	// Получаем заказ с полной информацией
	var device models.Device
	if err := db.Preload("Customer").Preload("Master").Preload("Client").Where("id = ?", deviceID).First(&device).Error; err != nil {
		log.Printf("Error getting device: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Заказ не найден")
		return
	}

	// Форматируем информацию о заказе
	text := fmt.Sprintf("📋 *Детали заказа %s*\n\n", device.Code)
	text += fmt.Sprintf("📱 *Устройство:* %s %s %s\n", device.DeviceType, device.Brand, device.Model)
	text += fmt.Sprintf("❗ *Проблема:* %s\n\n", device.Problem)

	if device.Customer != nil {
		text += fmt.Sprintf("👤 *Клиент:* %s\n", device.Customer.Name)
		text += fmt.Sprintf("📞 *Телефон:* %s\n\n", device.Customer.Phone)
	}

	if device.Client != nil {
		text += fmt.Sprintf("🆔 *Заказчик:* %v %v\n", device.Client.FirstName, device.Client.LastName)
		text += fmt.Sprintf("📱 *Telegram:* @%v\n\n", device.Client.Username)
	}

	statusIcon := getStatusIcon(device.Status)
	statusTextStr := getStatusTextRu(string(device.Status))
	text += fmt.Sprintf("%s *Статус:* %s\n", statusIcon, statusTextStr)

	if device.Master != nil {
		text += fmt.Sprintf("👨‍🔧 *Мастер:* %v %v\n", device.Master.FirstName, device.Master.LastName)
	} else {
		text += "👨‍🔧 *Мастер:* Не назначен\n"
	}

	text += "\n💰 *Финансы:*\n"
	if device.RepairCost > 0 {
		text += fmt.Sprintf("🔧 Ремонт: %.2f сум\n", device.RepairCost)
	}
	if device.PartsCost > 0 {
		text += fmt.Sprintf("🔩 Запчасти: %.2f сум\n", device.PartsCost)
	}
	if device.TotalCost > 0 {
		text += fmt.Sprintf("💵 Итого: %.2f сум\n", device.TotalCost)
	}
	if device.IsPaid {
		text += "✅ *Оплачено*\n"
	} else {
		text += "❌ *Не оплачено*\n"
	}

	text += fmt.Sprintf("\n📅 *Создан:* %s\n", device.CreatedAt.Format("02.01.2006 15:04"))
	text += fmt.Sprintf("🔄 *Обновлен:* %s\n", device.UpdatedAt.Format("02.01.2006 15:04"))

	// Создаем клавиатуру управления
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Изменить статус", fmt.Sprintf("admin_change_status_%d", device.ID)),
			tgbotapi.NewInlineKeyboardButtonData("💰 Цены", fmt.Sprintf("pricing_device_%d", device.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨‍🔧 Назначить мастера", fmt.Sprintf("admin_assign_master_%d", device.ID)),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("admin_edit_order_%d", device.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", fmt.Sprintf("admin_order_details_%d", device.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ К списку", "admin_orders"),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

// HandleAdminOrdersPage обрабатывает пагинацию заказов
func HandleAdminOrdersPage(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) < 5 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	filterStatus := parts[3]
	page, err := strconv.Atoi(parts[4])
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный номер страницы")
		return
	}

	lang := langCache.Get(callbackQuery.From.ID)

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для просмотра заказов")
		return
	}

	// Получаем заказы с фильтром
	var devices []models.Device
	query := db.Preload("Customer").Preload("Master").Order("created_at DESC")

	if filterStatus != "all" {
		query = query.Where("status = ?", filterStatus)
	}

	if err := query.Find(&devices).Error; err != nil {
		log.Printf("Error getting devices: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения заказов")
		return
	}

	showAdminOrdersList(bot, callbackQuery.From.ID, devices, filterStatus, page, lang)
}

// HandleAdminChangeStatus показывает меню изменения статуса заказа
func HandleAdminChangeStatus(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 4 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID заказа")
		return
	}

	_ = langCache.Get(callbackQuery.From.ID) // lang не используется пока

	// Создаем клавиатуру со статусами
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕 Принят", fmt.Sprintf("admin_set_status_%d_received", deviceID)),
			tgbotapi.NewInlineKeyboardButtonData("🔧 В работе", fmt.Sprintf("admin_set_status_%d_inProgress", deviceID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏳ Ожидание запчастей", fmt.Sprintf("admin_set_status_%d_waitingParts", deviceID)),
			tgbotapi.NewInlineKeyboardButtonData("✅ Готов", fmt.Sprintf("admin_set_status_%d_ready", deviceID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Выдан", fmt.Sprintf("admin_set_status_%d_completed", deviceID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменен", fmt.Sprintf("admin_set_status_%d_cancelled", deviceID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("admin_order_details_%d", deviceID)),
		),
	)

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, "🔄 Выберите новый статус:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// HandleAdminSetStatus обрабатывает установку статуса заказа
func HandleAdminSetStatus(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 5 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID заказа")
		return
	}

	newStatus := models.DeviceStatus(parts[4])

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для изменения статуса")
		return
	}

	// Обновляем статус заказа
	var device models.Device
	if err := db.First(&device, deviceID).Error; err != nil {
		log.Printf("Error getting device: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Заказ не найден")
		return
	}

	oldStatus := device.Status
	device.Status = newStatus

	if err := db.Save(&device).Error; err != nil {
		log.Printf("Error updating device status: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка обновления статуса")
		return
	}

	// Уведомляем клиента о изменении статуса
	if device.Client != nil {
		_ = langCache.Get(device.Client.TelegramID) // clientLang не используется пока
		notifyText := fmt.Sprintf("🔔 Статус вашего заказа %s изменен:\n\n%s ➡️ %s",
			device.Code,
			getStatusTextRu(string(oldStatus)),
			getStatusTextRu(string(newStatus)))

		clientMsg := tgbotapi.NewMessage(device.Client.TelegramID, notifyText)
		if _, err := bot.Send(clientMsg); err != nil {
			log.Printf("Error sending notification to client: %v", err)
		}
	}

	// Уведомляем мастера о изменении статуса (если назначен)
	if device.Master != nil {
		_ = langCache.Get(device.Master.TelegramID) // masterLang не используется пока
		notifyText := fmt.Sprintf("🔔 Статус заказа %s изменен администратором:\n\n%s ➡️ %s",
			device.Code,
			getStatusTextRu(string(oldStatus)),
			getStatusTextRu(string(newStatus)))

		masterMsg := tgbotapi.NewMessage(device.Master.TelegramID, notifyText)
		if _, err := bot.Send(masterMsg); err != nil {
			log.Printf("Error sending notification to master: %v", err)
		}
	}

	msg := tgbotapi.NewMessage(callbackQuery.From.ID,
		fmt.Sprintf("✅ Статус заказа %s изменен на: %s", device.Code, getStatusTextRu(string(newStatus))))
	bot.Send(msg)

	// Возвращаемся к деталям заказа
	callbackQuery.Data = fmt.Sprintf("admin_order_details_%d", deviceID)
	HandleAdminOrderDetails(bot, callbackQuery, db, langCache)
}

// HandleAdminAssignMaster показывает список мастеров для назначения
func HandleAdminAssignMaster(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 4 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID заказа")
		return
	}

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для назначения мастера")
		return
	}

	// Получаем список активных мастеров
	var masters []models.User
	if err := db.Where("role = ? AND is_active = ?", models.UserRoleMaster, true).Find(&masters).Error; err != nil {
		log.Printf("Error getting masters: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка получения мастеров")
		return
	}

	if len(masters) == 0 {
		msg := tgbotapi.NewMessage(callbackQuery.From.ID, "📋 Нет доступных мастеров")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("admin_order_details_%d", deviceID)),
			),
		)
		bot.Send(msg)
		return
	}

	// Создаем клавиатуру с мастерами
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, master := range masters {
		masterText := fmt.Sprintf("👨‍🔧 %v %v", master.FirstName, master.LastName)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(masterText, fmt.Sprintf("admin_do_assign_%d_%d", deviceID, master.ID)),
		})
	}

	// Кнопка отмены
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("admin_order_details_%d", deviceID)),
	})

	msg := tgbotapi.NewMessage(callbackQuery.From.ID, "👨‍🔧 Выберите мастера для назначения:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	bot.Send(msg)
}

// HandleAdminDoAssign обрабатывает назначение мастера на заказ
func HandleAdminDoAssign(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB, langCache *i18n.LanguageCache) {
	parts := strings.Split(callbackQuery.Data, "_")
	if len(parts) != 5 {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный формат данных")
		return
	}

	deviceID, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID заказа")
		return
	}

	masterID, err := strconv.ParseUint(parts[4], 10, 64)
	if err != nil {
		SendErrorMessage(bot, callbackQuery.From.ID, "Неверный ID мастера")
		return
	}

	var user models.User
	if err := db.First(&user, "telegram_id = ?", callbackQuery.From.ID).Error; err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if user.Role != models.UserRoleAdmin {
		SendErrorMessage(bot, callbackQuery.From.ID, "У вас нет прав для назначения мастера")
		return
	}

	// Получаем заказ и мастера
	var device models.Device
	if err := db.Preload("Client").First(&device, deviceID).Error; err != nil {
		log.Printf("Error getting device: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Заказ не найден")
		return
	}

	var master models.User
	if err := db.First(&master, masterID).Error; err != nil {
		log.Printf("Error getting master: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Мастер не найден")
		return
	}

	// Назначаем мастера
	masterIDUint := uint(masterID)
	device.MasterID = &masterIDUint

	// Если заказ был в статусе "Принят", меняем на "В работе"
	if device.Status == models.DeviceStatusReceived {
		device.Status = models.DeviceStatusInProgress
	}

	if err := db.Save(&device).Error; err != nil {
		log.Printf("Error assigning master: %v", err)
		SendErrorMessage(bot, callbackQuery.From.ID, "Ошибка назначения мастера")
		return
	}

	// Уведомляем мастера
	_ = langCache.Get(master.TelegramID) // masterLang не используется пока
	notifyText := fmt.Sprintf("🔔 Вам назначен новый заказ:\n\n📋 %s\n📱 %s %s %s\n❗ %s",
		device.Code, device.DeviceType, device.Brand, device.Model, device.Problem)

	masterMsg := tgbotapi.NewMessage(master.TelegramID, notifyText)
	if _, err := bot.Send(masterMsg); err != nil {
		log.Printf("Error sending notification to master: %v", err)
	}

	// Уведомляем клиента
	if device.Client != nil {
		_ = langCache.Get(device.Client.TelegramID) // clientLang не используется пока
		notifyText := fmt.Sprintf("🔔 На ваш заказ %v назначен мастер:\n\n👨‍🔧 %v %v",
			device.Code, master.FirstName, master.LastName)

		clientMsg := tgbotapi.NewMessage(device.Client.TelegramID, notifyText)
		if _, err := bot.Send(clientMsg); err != nil {
			log.Printf("Error sending notification to client: %v", err)
		}
	}

	msg := tgbotapi.NewMessage(callbackQuery.From.ID,
		fmt.Sprintf("✅ Мастер %v %v назначен на заказ %v", master.FirstName, master.LastName, device.Code))
	bot.Send(msg)

	// Возвращаемся к деталям заказа
	callbackQuery.Data = fmt.Sprintf("admin_order_details_%d", deviceID)
	HandleAdminOrderDetails(bot, callbackQuery, db, langCache)
}

// getStatusIcon возвращает иконку для статуса
func getStatusIcon(status models.DeviceStatus) string {
	switch status {
	case models.DeviceStatusReceived:
		return "🆕"
	case models.DeviceStatusInProgress:
		return "🔧"
	case models.DeviceStatusWaitingParts:
		return "⏳"
	case models.DeviceStatusReady:
		return "✅"
	case models.DeviceStatusCompleted:
		return "📦"
	case models.DeviceStatusCancelled:
		return "❌"
	default:
		return "❓"
	}
}

// getStatusTextRu возвращает текст статуса на русском
func getStatusTextRu(status string) string {
	switch status {
	case "received":
		return "🆕 Принят"
	case "inProgress":
		return "🔧 В работе"
	case "waitingParts":
		return "⏳ Ожидание запчастей"
	case "ready":
		return "✅ Готов"
	case "completed":
		return "📦 Выдан"
	case "cancelled":
		return "❌ Отменен"
	case "all":
		return "📋 Все заказы"
	default:
		return status
	}
}
