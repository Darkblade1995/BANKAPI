package main

import (
	"BANKAPI/internal/cache"
	"BANKAPI/internal/currency"
	"BANKAPI/internal/database"
	"BANKAPI/internal/env"
	"BANKAPI/internal/mailer"
	ws "BANKAPI/internal/websocket"
	"context"
	"log"

	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload"
)

type application struct {
	port      int
	jwtSecret string
	db        *database.Models
	converter *currency.Converter
	logger    *zap.Logger
	mailer    *mailer.Mailer
	cache     *cache.Cache
	hub       *ws.Hub
}

// @title           BankAPI
// @version         1.0
// @description     API bancaria con soporte multi-moneda, JWT y transferencias atómicas
// @termsOfService  http://swagger.io/terms/

// @contact.name   Fernando Agamez
// @contact.email  luisagamez050@gmail.com

// @host      localhost:8080
// @BasePath  /v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Escribe "Bearer" seguido de tu token JWT

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("failed to create logger:", err)
	}
	defer logger.Sync()

	db, err := database.NewConnection(database.Config{
		Host:     env.GetEnvString("DB_HOST", "localhost"),
		Port:     env.GetEnvInt("DB_PORT", 5432),
		User:     env.GetEnvString("DB_USER", "bankapi"),
		Password: env.GetEnvString("DB_PASSWORD", "bankapi123"),
		DBName:   env.GetEnvString("DB_NAME", "bankapi"),
	})
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	redisCache := cache.NewCache(
		env.GetEnvString("REDIS_ADDR", "localhost:6380"),
	)

	if err := redisCache.Ping(context.Background()); err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}

	logger.Info("connected to redis successfully")

	models := database.NewModels(db)

	converter := currency.NewConverter(
		env.GetEnvString("EXCHANGE_API_KEY", ""),
		redisCache,
	)

	m := mailer.NewMailer(
		env.GetEnvString("RESEND_API_KEY", ""),
		env.GetEnvString("RESEND_FROM", "onboarding@resend.dev"),
	)

	hub := ws.NewHub()
	go hub.Run()

	app := &application{
		port:      env.GetEnvInt("PORT", 8080),
		jwtSecret: env.GetEnvString("JWT_SECRET", "some-secret-123456"),
		db:        &models,
		converter: converter,
		logger:    logger,
		mailer:    m,
		cache:     redisCache,
		hub:       hub,
	}

	if err := app.serve(); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}