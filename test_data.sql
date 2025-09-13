-- Тестовые данные для системы iDoctor Bot

-- Очищаем существующие данные (будьте осторожны в продакшене)
-- DELETE FROM device_parts;
-- DELETE FROM parts;
-- DELETE FROM devices;
-- DELETE FROM customers;
-- DELETE FROM user_states;
-- DELETE FROM users;

-- Добавляем тестовых пользователей (админ и мастера)
INSERT INTO users (telegram_id, name, first_name, last_name, username, role, language, is_active, created_at, updated_at) VALUES
(123456789, 'Admin Test', 'Admin', 'User', 'admin_test', 'admin', 'ru', true, NOW(), NOW()),
(833391285, 'Мастер Иван', 'Иван', 'Петров', 'master_ivan', 'master', 'ru', true, NOW(), NOW()),
(7233051530, 'Мастер Петр', 'Петр', 'Сидоров', 'master_petr', 'master', 'ru', true, NOW(), NOW()),
(987654321, 'Мастер Ахмед', 'Ахмед', 'Каримов', 'master_ahmed', 'master', 'uz', false, NOW(), NOW())
ON CONFLICT (telegram_id) DO UPDATE SET
name = EXCLUDED.name,
first_name = EXCLUDED.first_name,
last_name = EXCLUDED.last_name,
username = EXCLUDED.username,
role = EXCLUDED.role,
language = EXCLUDED.language,
is_active = EXCLUDED.is_active,
updated_at = NOW();

-- Добавляем тестовых клиентов
INSERT INTO customers (name, phone, address, email, notes, created_at, updated_at) VALUES
('Иван Иванов', '+998901234567', 'ул. Навои, 15', 'ivan@example.com', 'Постоянный клиент', NOW(), NOW()),
('Мария Петрова', '+998907654321', 'пр. Амира Темура, 45', 'maria@example.com', '', NOW(), NOW()),
('Алексей Сидоров', '+998903456789', 'ул. Шота Руставели, 23', 'alex@example.com', 'Звонить после 18:00', NOW(), NOW()),
('Фатима Каримова', '+998909876543', 'ул. Бобура, 12', '', 'Говорит только на узбекском', NOW(), NOW())
ON CONFLICT (phone) DO UPDATE SET
name = EXCLUDED.name,
address = EXCLUDED.address,
email = EXCLUDED.email,
notes = EXCLUDED.notes,
updated_at = NOW();

-- Добавляем тестовые устройства
INSERT INTO devices (code, customer_id, master_id, device_type, brand, model, serial_number, problem, status, diagnosis, repair_cost, parts_cost, total_cost, is_paid, notes, received_at, deadline_at, warranty_days, created_at, updated_at) VALUES
('DEV001', 1, 1, 'Смартфон', 'Apple', 'iPhone 12', 'ABC123456789', 'Не включается, попадал в воду', 'inProgress', 'Требуется замена материнской платы', 120000, 30000, 150000, false, 'Клиент согласен на ремонт', NOW() - INTERVAL '2 days', NOW() + INTERVAL '5 days', 30, NOW(), NOW()),
('DEV002', 2, 2, 'Смартфон', 'Samsung', 'Galaxy S21', 'DEF987654321', 'Разбитый экран', 'ready', 'Заменен экран', 60000, 15000, 75000, true, 'Готов к выдаче', NOW() - INTERVAL '1 day', NOW() + INTERVAL '1 day', 14, NOW(), NOW()),
('DEV003', 3, NULL, 'Смартфон', 'Apple', 'iPhone 13', 'GHI111222333', 'Батарея быстро разряжается', 'received', '', 0, 0, 0, false, 'Ожидает назначения мастера', NOW(), NOW() + INTERVAL '7 days', 0, NOW(), NOW()),
('DEV004', 4, 1, 'Ноутбук', 'ASUS', 'VivoBook 15', 'JKL444555666', 'Не включается', 'waitingParts', 'Неисправен блок питания', 80000, 45000, 125000, false, 'Ждем поставку блока питания', NOW() - INTERVAL '3 days', NOW() + INTERVAL '10 days', 30, NOW(), NOW()),
('DEV005', 1, 2, 'Планшет', 'Apple', 'iPad Air', 'MNO777888999', 'Не работает Wi-Fi', 'completed', 'Перепрошивка и настройка', 25000, 0, 25000, true, 'Выдан клиенту', NOW() - INTERVAL '5 days', NOW() - INTERVAL '2 days', 14, NOW(), NOW());

-- Добавляем тестовые запчасти
INSERT INTO parts (name, brand, model, category, price, quantity, min_quantity, supplier, notes, created_at, updated_at) VALUES
('Экран iPhone 12', 'Apple', 'iPhone 12', 'Экраны', 25000, 5, 2, 'iStore', 'Оригинальные запчасти', NOW(), NOW()),
('Экран Samsung S21', 'Samsung', 'Galaxy S21', 'Экраны', 15000, 8, 3, 'Samsung Service', '', NOW(), NOW()),
('Батарея iPhone 13', 'Apple', 'iPhone 13', 'Батареи', 12000, 10, 5, 'iStore', '', NOW(), NOW()),
('Блок питания ASUS 65W', 'ASUS', 'Universal', 'Блоки питания', 45000, 2, 1, 'ASUS Service', 'Универсальный блок', NOW(), NOW())
ON CONFLICT (name, brand, model) DO UPDATE SET
price = EXCLUDED.price,
quantity = EXCLUDED.quantity,
min_quantity = EXCLUDED.min_quantity,
supplier = EXCLUDED.supplier,
notes = EXCLUDED.notes,
updated_at = NOW();

-- Связываем устройства с запчастями
INSERT INTO device_parts (device_id, part_id, quantity, used_at) VALUES
(1, 1, 1, NULL), -- iPhone 12 с экраном (еще не использован)
(2, 2, 1, NOW() - INTERVAL '1 day'), -- Samsung S21 с экраном (уже использован)
(4, 4, 1, NULL); -- ASUS с блоком питания (ожидает поставки)

-- Добавляем начальные состояния пользователей (все в состоянии idle)
INSERT INTO user_states (telegram_id, state, state_data, created_at, updated_at) VALUES
(123456789, 'idle', '{}', NOW(), NOW()),
(833391285, 'idle', '{}', NOW(), NOW()),
(7233051530, 'idle', '{}', NOW(), NOW()),
(987654321, 'idle', '{}', NOW(), NOW())
ON CONFLICT (telegram_id) DO UPDATE SET
state = EXCLUDED.state,
state_data = EXCLUDED.state_data,
updated_at = NOW();