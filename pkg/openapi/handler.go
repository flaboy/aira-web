package openapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/flaboy/aira-web/pkg/openapi/interfaces"

	"github.com/flaboy/pin"
	"github.com/gin-gonic/gin"
)

func HandleRequest(c *pin.Context) error {
	endpointName := c.Param("endpoint")
	return HandleEndpointRequest(c, interfaces.EndpointType(endpointName))
}

// HandleEndpointRequest 处理绑定到指定端点的请求。
func HandleEndpointRequest(c *pin.Context, endpointType interfaces.EndpointType) error {
	endpoint := findEndpoint(endpointType)
	if endpoint == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return nil
	}

	// 检查认证
	if err := endpoint.checkAuth(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "openapi.invalid_credentials", "message": "Invalid credentials or inactive application"}})
		return nil
	}
	allowed, err := appRepo.AllowRequest(c.Request.Context(), c.MustGet("application_database_id").(uint))
	if err != nil {
		return err
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "openapi.rate_limited", "message": "Application request limit exceeded"}})
		return nil
	}

	return endpoint.HandleApiRequest(c)
}

func (e *Endpoint) checkAuth(c *pin.Context) error {
	// 检查仓储是否已初始化
	if appRepo == nil {
		return errors.New("application repository not initialized")
	}

	// 从请求头获取认证信息
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return errors.New("missing authorization header")
	}

	// 只支持Basic Authentication
	// Authorization: Basic base64(client_id:client_secret)
	if !strings.HasPrefix(authHeader, "Basic ") {
		return errors.New("only basic authentication is supported")
	}

	// 使用Gin的内置解析Basic Auth
	clientID, clientSecret, ok := c.Request.BasicAuth()
	if !ok {
		return errors.New("invalid basic auth format")
	}

	// 验证应用是否存在且状态为active
	app, err := appRepo.FindByCredentials(clientID, clientSecret, e.Name, "active")
	if err != nil {
		return errors.New("invalid credentials or inactive application")
	}

	// 将应用信息存储到上下文中，供后续处理使用
	c.Set("application", app)
	c.Set("application_id", app.GetID())
	c.Set("application_database_id", app.GetDatabaseID())
	c.Set("application_user_id", app.GetUserID())
	c.Set("application_shop_id", app.GetShopID())
	c.Set("application_business_profile_id", app.GetBusinessProfileID())
	c.Set("application_scopes", app.GetScopes())
	if err := app.UpdateLastUsed(); err != nil {
		return err
	}

	return nil
}
