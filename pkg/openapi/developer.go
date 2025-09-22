package openapi

import (
	"strings"

	"github.com/flaboy/aira-web/pkg/openapi/interfaces"

	"github.com/flaboy/pin"
	"github.com/flaboy/pin/usererrors"
)

// 全局开发者服务实例
var developerService interfaces.DeveloperService

// SetDeveloperService 设置开发者服务
func SetDeveloperService(service interfaces.DeveloperService) {
	developerService = service
}

// GetDeveloperService 获取开发者服务
func GetDeveloperService() interfaces.DeveloperService {
	return developerService
}

// 全局开发者API处理器实例
var developerAPIHandler *DeveloperAPIHandler

// init 初始化开发者API处理器
func init() {
	developerAPIHandler = NewDeveloperAPIHandler()
}

// HandleDeveloperRequest 处理开发者相关请求的统一入口（使用简化的路由处理器）
func HandleDeveloperRequest(c *pin.Context, endpointType interfaces.EndpointType, service interfaces.DeveloperService, path string, userID uint) error {
	method := c.Request.Method

	// 使用新的简化处理器
	return developerAPIHandler.HandleRequest(c, path, method, service, userID)
}

// HandleDeveloperRequestLegacy 原有的处理开发者相关请求的统一入口（保留作为备用）
func HandleDeveloperRequestLegacy(c *pin.Context, endpointType interfaces.EndpointType, service interfaces.DeveloperService, path string, userID uint) error {
	method := c.Request.Method

	endpoint := GetEndpoint(endpointType)

	return endpoint.HandleDeveloperAPI(c, path, method, service, userID)
}

func (e *Endpoint) HandleDeveloperAPI(c *pin.Context, path string, method string, service interfaces.DeveloperService, userID uint) error {

	// 清理path，移除前导斜杠
	path = strings.TrimPrefix(path, "/")

	// 根据path和method路由到具体的处理函数
	switch {
	case path == "apps" && method == "GET":
		return e.handleGetApps(c, service, userID)
	case path == "apps" && method == "POST":
		return e.handleCreateApp(c, service, userID)
	// Handle /apps/:id/event-subscriptions and related routes before generic /apps/:id
	case strings.Contains(path, "/event-subscriptions/") && method == "DELETE":
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			appID := parts[1]
			eventCode := parts[3]
			return e.handleUnsubscribeEvent(c, service, userID, appID, eventCode)
		}
	case strings.Contains(path, "/event-subscriptions") && method == "GET":
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			appID := parts[1]
			return e.handleGetEventSubscriptions(c, service, userID, appID)
		}
	case strings.Contains(path, "/event-subscriptions") && method == "POST":
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			appID := parts[1]
			return e.handleSubscribeEvent(c, service, userID, appID)
		}
	case strings.Contains(path, "/regenerate-secret") && method == "POST":
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			appID := parts[1]
			return e.handleRegenerateSecret(c, service, userID, appID)
		}
	case strings.Contains(path, "/notify-config") && method == "PUT":
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			appID := parts[1]
			return e.handleUpdateNotifyConfig(c, service, userID, appID)
		}
	case strings.Contains(path, "/test-notify") && method == "POST":
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			appID := parts[1]
			return e.handleTestNotify(c, service, userID, appID)
		}
	case strings.HasPrefix(path, "apps/") && method == "GET":
		appID := strings.TrimPrefix(path, "apps/")
		return e.handleGetApp(c, service, userID, appID)
	case strings.HasPrefix(path, "apps/") && method == "PUT":
		appID := strings.TrimPrefix(path, "apps/")
		return e.handleUpdateApp(c, service, userID, appID)
	case strings.HasPrefix(path, "apps/") && method == "DELETE":
		appID := strings.TrimPrefix(path, "apps/")
		return e.handleDeleteApp(c, service, userID, appID)
	case path == "api-docs" && method == "GET":
		return e.handleGetApiDocs(c, service)
	case path == "event-docs" && method == "GET":
		return e.handleGetEventDocs(c, service)
	case path == "aws-config" && method == "GET":
		return e.handleGetAWSConfig(c, service)
	case path == "send-test-event" && method == "POST":
		return e.handleSendTestEvent(c, service, userID)
	default:
		return usererrors.New("API endpoint not found")
	}

	return usererrors.New("API endpoint not found")
}

// 具体的处理函数
func (e *Endpoint) handleGetApps(c *pin.Context, service interfaces.DeveloperService, userID uint) error {
	apps, err := service.GetApplications(userID)
	if err != nil {
		return usererrors.New("Failed to get applications: " + err.Error())
	}
	return c.Render(map[string]interface{}{"items": apps})
}

func (e *Endpoint) handleCreateApp(c *pin.Context, service interfaces.DeveloperService, userID uint) error {
	var form struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.BindJSON(&form); err != nil {
		return usererrors.New("Invalid request body")
	}

	app, err := service.CreateApplication(userID, form.Name, form.Description)
	if err != nil {
		return usererrors.New("Failed to create application: " + err.Error())
	}
	return c.Render(app)
}

func (e *Endpoint) handleGetApp(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	app, err := service.GetApplication(appID, userID)
	if err != nil {
		return usererrors.New("Failed to get application: " + err.Error())
	}
	return c.Render(app)
}

func (e *Endpoint) handleUpdateApp(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	var form struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := c.BindJSON(&form); err != nil {
		return usererrors.New("Invalid request body")
	}

	app, err := service.UpdateApplication(appID, userID, form.Name, form.Description, form.Status)
	if err != nil {
		return usererrors.New("Failed to update application: " + err.Error())
	}
	return c.Render(app)
}

func (e *Endpoint) handleDeleteApp(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	err := service.DeleteApplication(appID, userID)
	if err != nil {
		return usererrors.New("Failed to delete application: " + err.Error())
	}
	return c.Render(map[string]interface{}{"message": "Application deleted successfully"})
}

func (e *Endpoint) handleRegenerateSecret(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	app, err := service.RegenerateSecret(appID, userID)
	if err != nil {
		return usererrors.New("Failed to regenerate secret: " + err.Error())
	}
	return c.Render(app)
}

func (e *Endpoint) handleUpdateNotifyConfig(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	var form struct {
		NotifyType string `json:"notify_type" binding:"required"`
		NotifyURL  string `json:"notify_url" binding:"required"`
	}
	if err := c.BindJSON(&form); err != nil {
		return usererrors.New("Invalid request body")
	}

	app, err := service.UpdateNotifyConfig(appID, userID, form.NotifyType, form.NotifyURL)
	if err != nil {
		return usererrors.New("Failed to update notify config: " + err.Error())
	}
	return c.Render(app)
}

func (e *Endpoint) handleTestNotify(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	var form struct {
		NotifyType string `json:"notify_type" binding:"required"`
		NotifyURL  string `json:"notify_url" binding:"required"`
	}
	if err := c.BindJSON(&form); err != nil {
		return usererrors.New("Invalid request body")
	}

	err := service.TestNotify(appID, userID, form.NotifyType, form.NotifyURL)
	if err != nil {
		return usererrors.New("Failed to test notify: " + err.Error())
	}
	return c.Render(map[string]interface{}{"message": "Test notification sent successfully"})
}

func (e *Endpoint) handleGetEventSubscriptions(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	subscriptions, err := service.GetEventSubscriptions(appID, userID)
	if err != nil {
		return usererrors.New("Failed to get event subscriptions: " + err.Error())
	}
	return c.Render(subscriptions)
}

func (e *Endpoint) handleSubscribeEvent(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string) error {
	var form struct {
		EventCode string `json:"event_code" binding:"required"`
	}
	if err := c.BindJSON(&form); err != nil {
		return usererrors.New("Invalid request body")
	}

	subscription, err := service.SubscribeEvent(appID, userID, form.EventCode)
	if err != nil {
		return usererrors.New("Failed to subscribe event: " + err.Error())
	}
	return c.Render(subscription)
}

func (e *Endpoint) handleUnsubscribeEvent(c *pin.Context, service interfaces.DeveloperService, userID uint, appID string, eventCode string) error {
	err := service.UnsubscribeEvent(appID, userID, eventCode)
	if err != nil {
		return usererrors.New("Failed to unsubscribe event: " + err.Error())
	}
	return c.Render(map[string]interface{}{"message": "Event unsubscribed successfully"})
}

func (e *Endpoint) handleGetApiDocs(c *pin.Context, service interfaces.DeveloperService) error {
	docs, err := service.GetApiDocs()
	if err != nil {
		return usererrors.New("Failed to get API docs: " + err.Error())
	}
	return c.Render(docs)
}

func (e *Endpoint) handleGetEventDocs(c *pin.Context, service interfaces.DeveloperService) error {
	docs, err := service.GetEventDocs()
	if err != nil {
		return usererrors.New("Failed to get event docs: " + err.Error())
	}
	return c.Render(docs)
}

func (e *Endpoint) handleGetAWSConfig(c *pin.Context, service interfaces.DeveloperService) error {
	config, err := service.GetAWSConfig()
	if err != nil {
		return usererrors.New("Failed to get AWS config: " + err.Error())
	}
	return c.Render(config)
}

func (e *Endpoint) handleSendTestEvent(c *pin.Context, service interfaces.DeveloperService, userID uint) error {
	var form struct {
		AppID      string      `json:"app_id" binding:"required"`
		EventCode  string      `json:"event_code" binding:"required"`
		NotifyType string      `json:"notify_type" binding:"required"`
		NotifyURL  string      `json:"notify_url" binding:"required"`
		TestData   interface{} `json:"test_data"`
	}
	if err := c.BindJSON(&form); err != nil {
		return usererrors.New("Invalid request body")
	}

	err := service.SendTestEvent(form.AppID, userID, form.EventCode, form.NotifyType, form.NotifyURL, form.TestData)
	if err != nil {
		return usererrors.New("Failed to send test event: " + err.Error())
	}
	return c.Render(map[string]interface{}{"message": "Test event sent successfully"})
}
