package openapi

import (
	"github.com/flaboy/aira/aira-web/pkg/openapi/interfaces"
	"github.com/flaboy/aira/aira-web/pkg/routes"

	"github.com/flaboy/pin"
	"github.com/flaboy/pin/usererrors"
)

// DeveloperAPIHandler 开发者API处理器，使用简化的路由方式
type DeveloperAPIHandler struct {
	router *routes.GinRouter
}

// NewDeveloperAPIHandler 创建开发者API处理器
func NewDeveloperAPIHandler() *DeveloperAPIHandler {
	handler := &DeveloperAPIHandler{
		router: routes.NewGinRouter(""),
	}
	handler.registerRoutes()
	return handler
}

// registerRoutes 注册开发者API路由
func (h *DeveloperAPIHandler) registerRoutes() {
	// 应用管理路由
	h.router.GET("/apps", h.handleGetApps)
	h.router.POST("/apps", h.handleCreateApp)
	h.router.GET("/apps/:id", h.handleGetApp)
	h.router.PUT("/apps/:id", h.handleUpdateApp)
	h.router.DELETE("/apps/:id", h.handleDeleteApp)

	// 应用配置路由
	h.router.POST("/apps/:id/regenerate-secret", h.handleRegenerateSecret)
	h.router.PUT("/apps/:id/notify-config", h.handleUpdateNotifyConfig)
	h.router.POST("/apps/:id/test-notify", h.handleTestNotify)

	// 事件订阅路由
	h.router.GET("/apps/:id/event-subscriptions", h.handleGetEventSubscriptions)
	h.router.POST("/apps/:id/event-subscriptions", h.handleSubscribeEvent)
	h.router.DELETE("/apps/:id/event-subscriptions/:event_code", h.handleUnsubscribeEvent)

	// 文档和配置路由
	h.router.GET("/api-docs", h.handleGetApiDocs)
	h.router.GET("/event-docs", h.handleGetEventDocs)
	h.router.GET("/aws-config", h.handleGetAWSConfig)
	h.router.POST("/send-test-event", h.handleSendTestEvent)
}

// HandleRequest 处理请求的统一入口
func (h *DeveloperAPIHandler) HandleRequest(c *pin.Context, path, method string, service interfaces.DeveloperService, userID uint) error {
	// 将service和userID存储到context中，供处理器使用
	c.Set("developer_service", service)
	c.Set("user_id", userID)

	return h.router.HandleRequest(c, method, path)
}

// 以下是具体的处理器方法

func (h *DeveloperAPIHandler) handleGetApps(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)

	apps, err := service.GetApplications(userID)
	if err != nil {
		return usererrors.New("Failed to get applications: " + err.Error())
	}
	return c.Render(map[string]interface{}{"items": apps})
}

func (h *DeveloperAPIHandler) handleCreateApp(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)

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

func (h *DeveloperAPIHandler) handleGetApp(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

	app, err := service.GetApplication(appID, userID)
	if err != nil {
		return usererrors.New("Failed to get application: " + err.Error())
	}
	return c.Render(app)
}

func (h *DeveloperAPIHandler) handleUpdateApp(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

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

func (h *DeveloperAPIHandler) handleDeleteApp(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

	err := service.DeleteApplication(appID, userID)
	if err != nil {
		return usererrors.New("Failed to delete application: " + err.Error())
	}
	return c.Render(map[string]interface{}{"message": "Application deleted successfully"})
}

func (h *DeveloperAPIHandler) handleRegenerateSecret(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

	app, err := service.RegenerateSecret(appID, userID)
	if err != nil {
		return usererrors.New("Failed to regenerate secret: " + err.Error())
	}
	return c.Render(app)
}

func (h *DeveloperAPIHandler) handleUpdateNotifyConfig(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

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

func (h *DeveloperAPIHandler) handleTestNotify(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

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

func (h *DeveloperAPIHandler) handleGetEventSubscriptions(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

	subscriptions, err := service.GetEventSubscriptions(appID, userID)
	if err != nil {
		return usererrors.New("Failed to get event subscriptions: " + err.Error())
	}
	return c.Render(subscriptions)
}

func (h *DeveloperAPIHandler) handleSubscribeEvent(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")

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

func (h *DeveloperAPIHandler) handleUnsubscribeEvent(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)
	appID := routes.GetParam(c, "id")
	eventCode := routes.GetParam(c, "event_code")

	err := service.UnsubscribeEvent(appID, userID, eventCode)
	if err != nil {
		return usererrors.New("Failed to unsubscribe event: " + err.Error())
	}
	return c.Render(map[string]interface{}{"message": "Event unsubscribed successfully"})
}

func (h *DeveloperAPIHandler) handleGetApiDocs(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)

	docs, err := service.GetApiDocs()
	if err != nil {
		return usererrors.New("Failed to get API docs: " + err.Error())
	}
	return c.Render(docs)
}

func (h *DeveloperAPIHandler) handleGetEventDocs(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)

	docs, err := service.GetEventDocs()
	if err != nil {
		return usererrors.New("Failed to get event docs: " + err.Error())
	}
	return c.Render(docs)
}

func (h *DeveloperAPIHandler) handleGetAWSConfig(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)

	config, err := service.GetAWSConfig()
	if err != nil {
		return usererrors.New("Failed to get AWS config: " + err.Error())
	}
	return c.Render(config)
}

func (h *DeveloperAPIHandler) handleSendTestEvent(c *pin.Context) error {
	service := c.MustGet("developer_service").(interfaces.DeveloperService)
	userID := c.MustGet("user_id").(uint)

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
