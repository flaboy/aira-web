package interfaces

// DeveloperService 开发者功能服务接口
type DeveloperService interface {
	// 应用管理
	GetApplications(userID uint) ([]ApplicationInfo, error)
	GetApplication(appID string, userID uint) (ApplicationInfo, error)
	CreateApplication(userID uint, name, description string) (ApplicationInfo, error)
	UpdateApplication(appID string, userID uint, name, description, status string) (ApplicationInfo, error)
	DeleteApplication(appID string, userID uint) error
	RegenerateSecret(appID string, userID uint) (ApplicationInfo, error)

	// 通知配置
	UpdateNotifyConfig(appID string, userID uint, notifyType, notifyURL string) (ApplicationInfo, error)
	TestNotify(appID string, userID uint, notifyType, notifyURL string) error

	// 事件订阅
	GetEventSubscriptions(appID string, userID uint) ([]EventSubscriptionInfo, error)
	SubscribeEvent(appID string, userID uint, eventCode string) (EventSubscriptionInfo, error)
	UnsubscribeEvent(appID string, userID uint, eventCode string) error

	// 文档和配置
	GetApiDocs() (interface{}, error)
	GetEventDocs() ([]EventInfo, error)
	GetAWSConfig() (interface{}, error)
	SendTestEvent(appID string, userID uint, eventCode, notifyType, notifyURL string, testData interface{}) error
}

// EventInfo 事件信息接口
type EventInfo interface {
	GetCode() string
	GetName() string
	GetDescription() string
	GetExample() interface{}
}
