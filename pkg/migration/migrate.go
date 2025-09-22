package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/flaboy/aira-core/pkg/database"
	"github.com/flaboy/aira-core/pkg/redis"
	"github.com/flaboy/aira-web/pkg/config"
)

func init() {
	defaultStorage := &DefaultDatabaseMigrationStorage{}
	lockProvider := &RedisLockProvider{}
	migrationManager = NewMigrationManager(defaultStorage, lockProvider)
}

// RedisLockProvider 基于Redis的分布式锁实现
type RedisLockProvider struct{}

func (r *RedisLockProvider) Lock(key string, seconds int) (bool, error) {
	ctx := context.Background()
	result := redis.RedisClient.SetNX(ctx, key, "locked", time.Duration(seconds)*time.Second)
	return result.Val(), result.Err()
}

func (r *RedisLockProvider) Unlock(key string) error {
	ctx := context.Background()
	return redis.RedisClient.Del(ctx, key).Err()
}

// MigrationLog 迁移日志模型
type MigrationLog struct {
	ID        uint   `gorm:"primaryKey"`
	Migration string `gorm:"size:120"`
	Namespace string `gorm:"size:120;default:'app'"`
	AppliedAt time.Time
	Logs      string `gorm:"type:text"`
	Success   bool
}

func (m *MigrationLog) TableName() string {
	return config.Config.AiraTablePreifix + "migration_logs"
}

// DefaultDatabaseMigrationStorage 默认的数据库存储实现
type DefaultDatabaseMigrationStorage struct{}

func (d *DefaultDatabaseMigrationStorage) GetAppliedMigrations() ([]string, error) {
	// 确保迁移日志表存在
	if err := database.Database().AutoMigrate(&MigrationLog{}); err != nil {
		return nil, err
	}

	var logs []MigrationLog
	err := database.Database().Where("success = ?", true).Find(&logs).Error
	if err != nil {
		return nil, err
	}

	var names []string
	for _, log := range logs {
		key := fmt.Sprintf("%s:%s", log.Namespace, log.Migration)
		names = append(names, key)
	}
	return names, nil
}

func (d *DefaultDatabaseMigrationStorage) MarkMigrationApplied(namespace, name string) error {
	// 确保迁移日志表存在
	if err := database.Database().AutoMigrate(&MigrationLog{}); err != nil {
		return err
	}

	log := MigrationLog{
		Migration: name,
		Namespace: namespace,
		AppliedAt: time.Now(),
		Success:   true,
		Logs:      "",
	}
	return database.Database().Create(&log).Error
}

func (d *DefaultDatabaseMigrationStorage) MarkMigrationFailed(namespace, name string, errorMsg string) error {
	// 确保迁移日志表存在
	if err := database.Database().AutoMigrate(&MigrationLog{}); err != nil {
		return err
	}

	log := MigrationLog{
		Migration: name,
		Namespace: namespace,
		AppliedAt: time.Now(),
		Success:   false,
		Logs:      errorMsg,
	}
	return database.Database().Create(&log).Error
}

func (d *DefaultDatabaseMigrationStorage) MarkMigrationSkipped(namespace, name string) error {
	// 确保迁移日志表存在
	if err := database.Database().AutoMigrate(&MigrationLog{}); err != nil {
		return err
	}

	log := MigrationLog{
		Migration: name,
		Namespace: namespace,
		AppliedAt: time.Now(),
		Success:   true,
		Logs:      "skip", // 标记为跳过
	}
	return database.Database().Create(&log).Error
}

// 全局迁移管理器实例
var migrationManager *MigrationManager

func SetMigrationManager(storage MigrationStorage) {
	if migrationManager == nil {
		lockProvider := &RedisLockProvider{}
		migrationManager = NewMigrationManager(storage, lockProvider)
	}
}

func Start() error {
	for _, model := range needAutoMigrations {
		if err := database.Database().AutoMigrate(model); err != nil {
			return err
		}
	}

	return migrationManager.RunMigrations()
}

func AddMigrateWithNamespace(namespace, name string, fn func(*Migration) error) {
	migrationManager.Register(namespace, name, fn)
}

// AddMigrate 注册迁移函数（保持向后兼容）
func AddMigrate(name string, fn func(*Migration) error) {
	AddMigrateWithNamespace("app", name, fn)
}
