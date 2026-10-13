package database

import (
	"os"
	"path/filepath"

	"anywaycms/backend/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return nil, err
	}
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Site{},
		&model.Domain{},
		&model.Category{},
		&model.Content{},
		&model.Menu{},
		&model.Setting{},
		&model.Visit{},
		&model.Asset{},
	); err != nil {
		return nil, err
	}
	if err := Seed(db); err != nil {
		return nil, err
	}
	if err := EnsureSettings(db); err != nil {
		return nil, err
	}
	if err := EnsureAdminPath(db); err != nil {
		return nil, err
	}
	return db, nil
}
