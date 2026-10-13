package database

import (
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
