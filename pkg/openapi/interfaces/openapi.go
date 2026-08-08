package interfaces

import "context"

// ApplicationInfo 应用信息接口
type ApplicationInfo interface {
	GetID() string
	GetDatabaseID() uint
	GetUserID() uint
	GetBusinessProfileID() uint
	GetScopes() []string
	GetClientID() string
	GetClientSecret() string
	GetStatus() string
	GetNotifyType() string
	GetNotifyURL() string
	UpdateLastUsed() error
}

// ApplicationRepository 应用仓储接口
type ApplicationRepository interface {
	FindByCredentials(clientID, clientSecret string, endpointType EndpointType, status string) (ApplicationInfo, error)
	AllowRequest(ctx context.Context, applicationID uint) (bool, error)
}

// EventSubscriptionInfo 事件订阅信息接口
type EventSubscriptionInfo interface {
	GetApplicationID() uint
	GetEventCode() string
	GetApplication() ApplicationInfo
}

// EventSubscriptionRepository 事件订阅仓储接口
type EventSubscriptionRepository interface {
	FindByEventCode(eventCode string) ([]EventSubscriptionInfo, error)
}

type EndpointType string

type NotifyType string

const (
	NotifyTypeWebhook NotifyType = "webhook"
	NotifyTypeSQS     NotifyType = "sqs"
)
