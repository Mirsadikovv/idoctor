package src

import (
	"git.sriss.uz/mehnat/dmi_bot/app/bot"
	"git.sriss.uz/mehnat/dmi_bot/app/config"
	"git.sriss.uz/shared/shared_service/pg"
)

type Env struct {
	POSTGRES_HOST      string `env:"POSTGRES_HOST"`
	POSTGRES_USER      string `env:"POSTGRES_USER"`
	POSTGRES_PASSWORD  string `env:"POSTGRES_PASSWORD"`
	POSTGRES_DB        string `env:"POSTGRES_DB"`
	POSTGRES_PORT      int    `env:"POSTGRES_PORT"`
	POSTGRES_SSL_MODE  string `env:"POSTGRES_SSL_MODE" default:"disable"`
	POSTGRES_TIME_ZONE string `env:"POSTGRES_TIME_ZONE" default:"UTC"`
}

func Exec(env *Env) {

	gormConfig := &pg.GormConfig{
		SkipDefaultTransaction: true,
	}

	pgConfig := pg.ConnectionConfig{
		Host:     env.POSTGRES_HOST,
		User:     env.POSTGRES_USER,
		Password: env.POSTGRES_PASSWORD,
		DBName:   env.POSTGRES_DB,
		Port:     env.POSTGRES_PORT,
		SSLMode:  env.POSTGRES_SSL_MODE,
		TimeZone: env.POSTGRES_TIME_ZONE,
	}

	db := pg.Primary(gormConfig, pgConfig)

	cfg, err := config.Load()
	if err != nil {
		return
	}

	bot.Start(cfg, db)

}
