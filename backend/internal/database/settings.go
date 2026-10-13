package database

import (
	"os"

	"anywaycms/backend/internal/adminpath"
	"anywaycms/backend/internal/model"

	"gorm.io/gorm"
)

func EnsureSettings(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.Setting{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	row := model.Setting{
		SystemName:   "站群 CMS",
		SystemNotice: "一套后台，多个网站",
		TrackVisits:  1,
		RetainDays:   90,
	}
	if err := db.Create(&row).Error; err != nil {
		return err
	}
	return db.Model(&row).Updates(map[string]any{
		"track_visits": 1,
		"retain_days":  90,
	}).Error
}

func EnsureAdminPath(db *gorm.DB) error {
	var row model.Setting
	if err := db.Order("id asc").First(&row).Error; err != nil {
		return err
	}
	if _, msg := adminpath.Normalize(row.AdminPath); msg == "" {
		return nil
	}
	next, msg := adminpath.Normalize(os.Getenv("ADMIN_PATH"))
	if msg != "" {
		next = adminpath.Random()
	}
	return db.Model(&row).Update("admin_path", next).Error
}
