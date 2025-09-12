package main

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"idoctor-bot/app/bot"
	"idoctor-bot/app/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки конфигурации:", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Ошибка получения SQL DB:", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnectionMaxLifetime)

	log.Println("Запуск Telegram бота для ремонтной мастерской...")

	if err := bot.Start(cfg, db); err != nil {
		log.Fatal("Ошибка запуска бота:", err)
	}
}
