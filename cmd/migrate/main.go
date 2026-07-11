package main

import (
	"BANKAPI/internal/database"
	"BANKAPI/internal/env"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please provide a migration direction: 'up' or 'down'")
	}

	direction := os.Args[1]

	db, err := database.NewConnection(database.Config{
		Host:     env.GetEnvString("DB_HOST", "localhost"),
		Port:     env.GetEnvInt("DB_PORT", 5432),
		User:     env.GetEnvString("DB_USER", "bankapi"),
		Password: env.GetEnvString("DB_PASSWORD", "bankapi123"),
		DBName:   env.GetEnvString("DB_NAME", "bankapi"),
	})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("failed to create migration driver:", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatal("failed to create migrate instance:", err)
	}

	if direction == "up" {
		err = m.Up()
	} else if direction == "down" {
		err = m.Down()
	} else {
		log.Fatal("Invalid direction. Use 'up' or 'down'")
	}

	if err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}

	log.Println("Migration", direction, "completed successfully")
}