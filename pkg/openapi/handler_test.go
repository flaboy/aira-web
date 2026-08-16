package openapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flaboy/aira-web/pkg/openapi/interfaces"
	"github.com/flaboy/pin"
	"github.com/flaboy/pin/usererrors"
	"github.com/gin-gonic/gin"
)

type handlerTestApplication struct {
	scopes []string
}

func (handlerTestApplication) GetID() string              { return "app_test" }
func (handlerTestApplication) GetDatabaseID() uint        { return 7 }
func (handlerTestApplication) GetUserID() uint            { return 11 }
func (handlerTestApplication) GetBusinessProfileID() uint { return 22 }
func (handlerTestApplication) GetShopID() uint            { return 33 }
func (a handlerTestApplication) GetScopes() []string      { return a.scopes }
func (handlerTestApplication) GetClientID() string        { return "client_test" }
func (handlerTestApplication) GetClientSecret() string    { return "" }
func (handlerTestApplication) GetStatus() string          { return "active" }
func (handlerTestApplication) GetNotifyType() string      { return "" }
func (handlerTestApplication) GetNotifyURL() string       { return "" }
func (handlerTestApplication) UpdateLastUsed() error      { return nil }

type handlerTestRepository struct {
	err         error
	rateLimited bool
}

func (r handlerTestRepository) FindByCredentials(string, string, interfaces.EndpointType, string) (interfaces.ApplicationInfo, error) {
	if r.err != nil {
		return nil, r.err
	}
	return handlerTestApplication{scopes: []string{"catalog:read"}}, nil
}

func (r handlerTestRepository) AllowRequest(context.Context, uint) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	return !r.rateLimited, nil
}

func TestHandleEndpointRequestStopsAfterAuthenticationFailure(t *testing.T) {
	endpointType := interfaces.EndpointType("handler-auth-failure")
	handlerCalls := 0
	RegisterGetApi(endpointType, "probe", func(c *pin.Context) (map[string]bool, *usererrors.Error) {
		handlerCalls++
		return map[string]bool{"ok": true}, nil
	}, "Probe")

	previousRepository := GetApplicationRepository()
	SetApplicationRepository(handlerTestRepository{err: errors.New("invalid credentials")})
	defer SetApplicationRepository(previousRepository)

	context, recorder := newHandlerTestContext(http.MethodGet, "/probe")
	context.Params = append(context.Params, gin.Param{Key: "path", Value: "/probe"})
	context.Request.SetBasicAuth("client_test", "secret_test")

	if err := HandleEndpointRequest(context, endpointType); err != nil {
		t.Fatalf("处理认证失败请求：%v", err)
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("状态码=%d，期望=%d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerCalls != 0 {
		t.Fatalf("认证失败后业务 Handler 被调用 %d 次", handlerCalls)
	}
}

func TestHandleEndpointRequestStopsWhenApplicationIsRateLimited(t *testing.T) {
	endpointType := interfaces.EndpointType("handler-rate-limited")
	handlerCalls := 0
	RegisterGetApi(endpointType, "probe", func(c *pin.Context) (map[string]bool, *usererrors.Error) {
		handlerCalls++
		return map[string]bool{"ok": true}, nil
	}, "Probe")

	previousRepository := GetApplicationRepository()
	SetApplicationRepository(handlerTestRepository{rateLimited: true})
	defer SetApplicationRepository(previousRepository)

	context, recorder := newHandlerTestContext(http.MethodGet, "/probe")
	context.Params = append(context.Params, gin.Param{Key: "path", Value: "/probe"})
	context.Request.SetBasicAuth("client_test", "secret_test")
	if err := HandleEndpointRequest(context, endpointType); err != nil {
		t.Fatalf("处理限流请求：%v", err)
	}
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d，期望=%d", recorder.Code, http.StatusTooManyRequests)
	}
	if handlerCalls != 0 {
		t.Fatalf("限流后仍执行了业务处理器 %d 次", handlerCalls)
	}
}

func TestHandleApiRequestRejectsMissingScope(t *testing.T) {
	endpointType := interfaces.EndpointType("handler-scope-denied")
	handlerCalls := 0
	RegisterGetApi(endpointType, "products", func(c *pin.Context) (map[string]bool, *usererrors.Error) {
		handlerCalls++
		return map[string]bool{"ok": true}, nil
	}, "Products").WithScope("orders:read")

	previousRepository := GetApplicationRepository()
	SetApplicationRepository(handlerTestRepository{})
	defer SetApplicationRepository(previousRepository)

	context, _ := newHandlerTestContext(http.MethodGet, "/products")
	context.Params = append(context.Params, gin.Param{Key: "path", Value: "/products"})
	context.Request.SetBasicAuth("client_test", "secret_test")

	err := HandleEndpointRequest(context, endpointType)
	scopeErr, ok := err.(*usererrors.Error)
	if !ok || scopeErr.Code() != "openapi.scope_denied" {
		t.Fatalf("权限错误=%v", err)
	}
	if handlerCalls != 0 {
		t.Fatalf("权限不足后业务 Handler 被调用 %d 次", handlerCalls)
	}
}

func TestHandleApiRequestMatchesPathParameters(t *testing.T) {
	endpointType := interfaces.EndpointType("handler-path-parameter")
	RegisterGetApi(endpointType, "orders/:id", func(c *pin.Context) (map[string]string, *usererrors.Error) {
		return map[string]string{"id": c.Param("id")}, nil
	}, "Order")

	previousRepository := GetApplicationRepository()
	SetApplicationRepository(handlerTestRepository{})
	defer SetApplicationRepository(previousRepository)

	context, recorder := newHandlerTestContext(http.MethodGet, "/orders/EPO-1")
	context.Params = append(context.Params, gin.Param{Key: "path", Value: "/orders/EPO-1"})
	context.Request.SetBasicAuth("client_test", "secret_test")

	if err := HandleEndpointRequest(context, endpointType); err != nil {
		t.Fatalf("处理路径参数：%v", err)
	}
	if !strings.Contains(recorder.Body.String(), "EPO-1") {
		t.Fatalf("路径参数响应=%s", recorder.Body.String())
	}
}

func TestHandleEndpointRequestUsesExplicitEndpoint(t *testing.T) {
	endpointType := interfaces.EndpointType("handler-explicit-endpoint")
	handlerCalls := 0
	RegisterGetApi(endpointType, "probe", func(c *pin.Context) (map[string]bool, *usererrors.Error) {
		handlerCalls++
		return map[string]bool{"ok": true}, nil
	}, "Probe")

	previousRepository := GetApplicationRepository()
	SetApplicationRepository(handlerTestRepository{})
	defer SetApplicationRepository(previousRepository)

	context, recorder := newHandlerTestContext(http.MethodGet, "/probe")
	context.Params = append(context.Params, gin.Param{Key: "path", Value: "/probe"})
	context.Request.SetBasicAuth("client_test", "secret_test")

	if err := HandleEndpointRequest(context, endpointType); err != nil {
		t.Fatalf("处理显式端点请求：%v", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码=%d，期望=%d", recorder.Code, http.StatusOK)
	}
	if handlerCalls != 1 {
		t.Fatalf("业务 Handler 调用次数=%d，期望=1", handlerCalls)
	}
}

func newHandlerTestContext(method, path string) (*pin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(method, path, nil)
	return &pin.Context{Context: ginContext}, recorder
}
