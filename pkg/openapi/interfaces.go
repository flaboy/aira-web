package openapi

import "github.com/flaboy/aira-web/pkg/openapi/interfaces"

// 全局仓储实例
var (
	appRepo   interfaces.ApplicationRepository
	eventRepo interfaces.EventSubscriptionRepository
)

// SetApplicationRepository 设置应用仓储
func SetApplicationRepository(repo interfaces.ApplicationRepository) {
	appRepo = repo
}

// SetEventSubscriptionRepository 设置事件订阅仓储
func SetEventSubscriptionRepository(repo interfaces.EventSubscriptionRepository) {
	eventRepo = repo
}

// GetApplicationRepository 获取应用仓储
func GetApplicationRepository() interfaces.ApplicationRepository {
	return appRepo
}

// GetEventSubscriptionRepository 获取事件订阅仓储
func GetEventSubscriptionRepository() interfaces.EventSubscriptionRepository {
	return eventRepo
}
