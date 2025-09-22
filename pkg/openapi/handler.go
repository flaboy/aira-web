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

	// 获取端点
	endpoint := GetEndpoint(interfaces.EndpointType(endpointName))
	if endpoint == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return nil
	}

	// 检查认证
	if err := endpoint.checkAuth(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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

	// 更新最后使用时间（异步执行，避免阻塞请求）
	go func() {
		app.UpdateLastUsed()
	}()

	// 将应用信息存储到上下文中，供后续处理使用
	c.Set("application", app)
	c.Set("application_id", app.GetID())

	return nil
}
