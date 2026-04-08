package db

import (
	"fmt"
	"os"
	"strings"

	"github.com/spaceballone/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init opens the database and runs auto-migrations.
// If DATABASE_URL starts with "postgres", it uses PostgreSQL; otherwise SQLite.
func Init() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "spaceballone.db"
	}

	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	var database *gorm.DB
	var err error

	if strings.HasPrefix(dsn, "postgres") {
		database, err = gorm.Open(postgres.Open(dsn), cfg)
	} else {
		// Strip sqlite:// prefix if present
		dsn = strings.TrimPrefix(dsn, "sqlite://")
		database, err = gorm.Open(sqlite.Open(dsn), cfg)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate all models
	if err := database.AutoMigrate(models.AllModels()...); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return database, nil
}
