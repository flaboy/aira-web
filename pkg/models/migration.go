package models

import (
	"fmt"
	"time"

	"github.com/flaboy/aira-core/pkg/database"
	"github.com/flaboy/aira-web/pkg/migration"
)

type MigrationLogs struct {
	ID        uint   `gorm:"primaryKey"`
	Migration string `gorm:"size:120"`
	Namespace string `gorm:"size:120"`
	AppliedAt time.Time
	Logs      string `gorm:"type:text"`
	Success   bool
}

// GormMigrationStorage 基于GORM的迁移存储实现
type GormMigrationStorage struct{}

func (g *GormMigrationStorage) GetAppliedMigrations() ([]string, error) {
	var logs []MigrationLogs
	err := database.Database().Find(&logs).Error
	if err != nil {
		return nil, err
	}

	var names []string
	for _, log := range logs {
		if log.Success {
			key := fmt.Sprintf("%s:%s", log.Namespace, log.Migration)
			names = append(names, key)
		}
	}
	return names, nil
}

func (g *GormMigrationStorage) MarkMigrationApplied(namespace, name string) error {
	log := MigrationLogs{
		Migration: name,
		Namespace: namespace,
		AppliedAt: time.Now(),
		Success:   true,
		Logs:      "",
	}
	return database.Database().Create(&log).Error
}

func (g *GormMigrationStorage) MarkMigrationSkipped(namespace, name string) error {
	log := MigrationLogs{
		Migration: name,
		Namespace: namespace,
		AppliedAt: time.Now(),
		Success:   true,
		Logs:      "skip", // 标记为跳过
	}
	return database.Database().Create(&log).Error
}

func (g *GormMigrationStorage) MarkMigrationFailed(namespace, name string, errorMsg string) error {
	log := MigrationLogs{
		Migration: name,
		Namespace: namespace,
		AppliedAt: time.Now(),
		Success:   false,
		Logs:      errorMsg,
	}
	return database.Database().Create(&log).Error
}

func init() {
	migration.RegisterAutoMigrateModels(&MigrationLogs{})
	migration.SetMigrationManager(&GormMigrationStorage{})
}
