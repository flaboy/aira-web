package migration

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// MigrationStorage 定义迁移状态存储接口
type MigrationStorage interface {
	GetAppliedMigrations() ([]string, error)
	MarkMigrationApplied(namespace, name string) error
	MarkMigrationFailed(namespace, name string, errorMsg string) error
	MarkMigrationSkipped(namespace, name string) error
}

// LockProvider 定义分布式锁接口
type LockProvider interface {
	Lock(key string, seconds int) (bool, error)
	Unlock(key string) error
}

// Migration 迁移执行上下文
type Migration struct {
	logStrings []string
	storage    MigrationStorage
}

func (m *Migration) Log(format string, args ...interface{}) {
	m.logStrings = append(m.logStrings, fmt.Sprintf(format, args...))
}

func (m *Migration) LogString() string {
	return strings.Join(m.logStrings, "\n")
}

// MigrationFunc 迁移函数类型
type MigrationFunc func(*Migration) error

// MigrationItem 迁移项
type MigrationItem struct {
	Namespace string
	Name      string
	Func      MigrationFunc
}

// MigrationManager 迁移管理器
type MigrationManager struct {
	storage      MigrationStorage
	lockProvider LockProvider
	migrations   []*MigrationItem // 使用切片保持顺序
}

func NewMigrationManager(storage MigrationStorage, lockProvider LockProvider) *MigrationManager {
	return &MigrationManager{
		storage:      storage,
		lockProvider: lockProvider,
		migrations:   make([]*MigrationItem, 0),
	}
}

func (m *MigrationManager) Register(namespace, name string, fn MigrationFunc) {
	m.migrations = append(m.migrations, &MigrationItem{
		Namespace: namespace,
		Name:      name,
		Func:      fn,
	})
}

func (m *MigrationManager) RunMigrations() error {
	slog.Info("RunMigrations", "count", len(m.migrations))
	const lockKey = "migrate_lock"
	const lockTimeout = 60

	// 获取分布式锁
	locked, err := m.lockProvider.Lock(lockKey, lockTimeout)
	if err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("migration is already running")
	}
	defer m.lockProvider.Unlock(lockKey)

	// 获取已应用的迁移
	appliedMigrations, err := m.storage.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedSet := make(map[string]bool)
	for _, name := range appliedMigrations {
		appliedSet[name] = true
	}

	// 按 namespace 分组检查是否为新环境
	namespaceHasRecords := make(map[string]bool)
	for _, name := range appliedMigrations {
		parts := strings.Split(name, ":")
		if len(parts) == 2 {
			namespaceHasRecords[parts[0]] = true
		}
	}

	// 对于新环境的 namespace，将所有迁移标记为跳过
	for _, item := range m.migrations {
		slog.Info("Processing migration", "name", item.Name, "namespace", item.Namespace)
		key := fmt.Sprintf("%s:%s", item.Namespace, item.Name)

		// 如果该 namespace 没有任何记录，说明是新环境
		if !namespaceHasRecords[item.Namespace] && !appliedSet[key] {
			err = m.storage.MarkMigrationSkipped(item.Namespace, item.Name)
			if err != nil {
				return fmt.Errorf("failed to mark migration %s as skipped: %w", key, err)
			}
			appliedSet[key] = true
			namespaceHasRecords[item.Namespace] = true
		}
	}

	// 执行未应用的迁移
	for _, item := range m.migrations {
		key := fmt.Sprintf("%s:%s", item.Namespace, item.Name)
		if appliedSet[key] {
			continue
		}

		migration := &Migration{
			storage: m.storage,
		}

		migration.Log("Starting migration: %s:%s at %s", item.Namespace, item.Name, time.Now().Format(time.RFC3339))

		err := item.Func(migration)
		if err != nil {
			errorMsg := fmt.Sprintf("Migration failed: %v\nLogs:\n%s", err, migration.LogString())
			m.storage.MarkMigrationFailed(item.Namespace, item.Name, errorMsg)
			return fmt.Errorf("migration %s:%s failed: %w", item.Namespace, item.Name, err)
		}

		migration.Log("Migration completed: %s:%s at %s", item.Namespace, item.Name, time.Now().Format(time.RFC3339))

		err = m.storage.MarkMigrationApplied(item.Namespace, item.Name)
		if err != nil {
			return fmt.Errorf("failed to mark migration %s:%s as applied: %w", item.Namespace, item.Name, err)
		}
	}

	return nil
}
