package openapi

import (
	"reflect"
	"strings"
	"sync"

	"github.com/flaboy/aira/aira-web/pkg/openapi/interfaces"

	"github.com/flaboy/pin"
	"github.com/flaboy/pin/usererrors"
)

type EventCode string

type EventInfo struct {
	Code        EventCode   `json:"code"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Object      interface{} `json:"example"` // 用于反射生成文档和Example数据
}

type Endpoint struct {
	Name      interfaces.EndpointType
	Events    map[EventCode]*EventInfo
	eventlist []*EventInfo
	apilist   []ApiRouter
	mutex     sync.RWMutex
}

var endpoints = make(map[interfaces.EndpointType]*Endpoint)
var endpointsMutex sync.RWMutex

func GetEndpoint(name interfaces.EndpointType) *Endpoint {
	endpointsMutex.Lock()
	defer endpointsMutex.Unlock()

	if ep, exists := endpoints[name]; exists {
		return ep
	}

	ep := &Endpoint{
		Name:    name,
		Events:  make(map[EventCode]*EventInfo),
		apilist: make([]ApiRouter, 0),
	}
	endpoints[name] = ep
	return ep
}

func (e *Endpoint) AddEvent(event EventInfo) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.Events[event.Code] = &event
	e.eventlist = append(e.eventlist, &event)
}

func (e *Endpoint) GetAllEvents() []*EventInfo {
	return e.eventlist
}

type ApiRouter struct {
	Name            string
	Method          string
	Path            string
	Handler         func(c *pin.Context, request any) (response any, err *usererrors.Error)
	Request         interface{}
	Response        interface{}
	Errors          []*usererrors.Error
	RequestExample  interface{} // 请求示例
	ResponseExample interface{} // 响应示例
}

// ApiBuilder 用于支持链式调用的API构建器
type ApiBuilder struct {
	registeredRouter *ApiRouter // 已注册的API路由的引用
}

// WithExample 设置请求示例
func (b *ApiBuilder) WithExample(example interface{}) *ApiBuilder {
	if b.registeredRouter != nil {
		b.registeredRouter.RequestExample = example
	}
	return b
}

// WithErrors sets error examples
func (b *ApiBuilder) WithErrors(errors ...*usererrors.Error) *ApiBuilder {
	if b.registeredRouter != nil {
		b.registeredRouter.Errors = append(b.registeredRouter.Errors, errors...)
	}
	return b
}

// WithResponseExample 设置响应示例
func (b *ApiBuilder) WithResponseExample(example interface{}) *ApiBuilder {
	if b.registeredRouter != nil {
		b.registeredRouter.ResponseExample = example
	}
	return b
}

// buildAndRegister 构建并注册API，返回ApiBuilder引用
func buildAndRegister(
	endpointType interfaces.EndpointType,
	method, path, apiName string,
	handler func(c *pin.Context, request any) (response any, err *usererrors.Error),
	request, response interface{},
	errors []*usererrors.Error,
) *ApiBuilder {
	router := ApiRouter{
		Name:            apiName,
		Method:          method,
		Path:            path,
		Handler:         handler,
		Request:         request,
		Response:        response,
		Errors:          errors,
		RequestExample:  nil, // 初始为nil，通过WithExample设置
		ResponseExample: nil, // 初始为nil，通过WithResponseExample设置
	}

	e := GetEndpoint(endpointType)
	e.apilist = append(e.apilist, router)

	// 返回ApiBuilder，引用刚刚添加的API
	return &ApiBuilder{
		registeredRouter: &e.apilist[len(e.apilist)-1],
	}
}

// 通用的处理器接口，用于类型安全的API处理
type ApiHandler interface {
	Handle(c *pin.Context, request any) (response any, err *usererrors.Error)
}

// 泛型处理器适配器
type TypedApiHandler[Req any, Resp any] struct {
	HandlerFunc func(c *pin.Context, request Req) (response Resp, err *usererrors.Error)
}

func (h *TypedApiHandler[Req, Resp]) Handle(c *pin.Context, request any) (response any, err *usererrors.Error) {
	typedRequest, ok := request.(Req)
	if !ok {
		return nil, usererrors.New("invalid_request_type", "Invalid request type")
	}
	return h.HandlerFunc(c, typedRequest)
}

// 无请求体的处理器适配器
type NoRequestApiHandler[Resp any] struct {
	HandlerFunc func(c *pin.Context) (response Resp, err *usererrors.Error)
}

func (h *NoRequestApiHandler[Resp]) Handle(c *pin.Context, request any) (response any, err *usererrors.Error) {
	return h.HandlerFunc(c)
}

// 注册类型安全的API路由 - 内部通用函数，返回ApiBuilder
func registerTypedApiRouter[Req any, Resp any](
	t interfaces.EndpointType,
	method, path string,
	handler func(c *pin.Context, request Req) (response Resp, err *usererrors.Error),
	apiName string,
	errors ...*usererrors.Error,
) *ApiBuilder {
	typedHandler := &TypedApiHandler[Req, Resp]{HandlerFunc: handler}

	// 通过反射获取类型信息（仅用于类型检查，不作为示例）
	var reqExample Req
	var respExample Resp

	// 如果是指针类型，创建指向零值的指针
	reqType := reflect.TypeOf(reqExample)
	if reqType != nil && reqType.Kind() == reflect.Ptr {
		reqExample = reflect.New(reqType.Elem()).Interface().(Req)
	}

	respType := reflect.TypeOf(respExample)
	if respType != nil && respType.Kind() == reflect.Ptr {
		respExample = reflect.New(respType.Elem()).Interface().(Resp)
	}

	return buildAndRegister(t, method, path, apiName, typedHandler.Handle, reqExample, respExample, errors)
}

// 注册无请求体的API路由 - 内部通用函数，返回ApiBuilder
func registerNoRequestRouter[Resp any](
	t interfaces.EndpointType,
	method, path string,
	handler func(c *pin.Context) (response Resp, err *usererrors.Error),
	apiName string,
	errors ...*usererrors.Error,
) *ApiBuilder {
	noRequestHandler := &NoRequestApiHandler[Resp]{HandlerFunc: handler}

	// 通过反射获取响应类型信息（仅用于类型检查，不作为示例）
	var respExample Resp
	respType := reflect.TypeOf(respExample)
	if respType != nil && respType.Kind() == reflect.Ptr {
		respExample = reflect.New(respType.Elem()).Interface().(Resp)
	}

	return buildAndRegister(t, method, path, apiName, noRequestHandler.Handle, nil, respExample, errors)
}

// HTTP方法特定的注册函数

// 注册POST API（需要请求体）
func RegisterPostApi[Req any, Resp any](
	t interfaces.EndpointType,
	path string,
	handler func(c *pin.Context, request Req) (response Resp, err *usererrors.Error),
	apiName string,
	errors ...*usererrors.Error,
) *ApiBuilder {
	return registerTypedApiRouter(t, "POST", path, handler, apiName, errors...)
}

// 注册PUT API（需要请求体）
func RegisterPutApi[Req any, Resp any](
	t interfaces.EndpointType,
	path string,
	handler func(c *pin.Context, request Req) (response Resp, err *usererrors.Error),
	apiName string,
	errors ...*usererrors.Error,
) *ApiBuilder {
	return registerTypedApiRouter(t, "PUT", path, handler, apiName, errors...)
}

// 注册GET API（无请求体）
func RegisterGetApi[Resp any](
	t interfaces.EndpointType,
	path string,
	handler func(c *pin.Context) (response Resp, err *usererrors.Error),
	apiName string,
	errors ...*usererrors.Error,
) *ApiBuilder {
	return registerNoRequestRouter(t, "GET", path, handler, apiName, errors...)
}

// 注册DELETE API（无请求体）
func RegisterDeleteApi[Resp any](
	t interfaces.EndpointType,
	path string,
	handler func(c *pin.Context) (response Resp, err *usererrors.Error),
	apiName string,
	errors ...*usererrors.Error,
) *ApiBuilder {
	return registerNoRequestRouter(t, "DELETE", path, handler, apiName, errors...)
}

// 获取所有注册的API路由
func (e *Endpoint) GetApiList() []ApiRouter {
	return e.apilist
}

// API请求处理器
func (e *Endpoint) HandleApiRequest(c *pin.Context) error {
	// 获取请求路径（去掉前导斜杠）
	path := strings.TrimPrefix(c.Param("path"), "/")
	method := c.Request.Method

	// 遍历已注册的API路由
	for _, router := range e.apilist {
		if router.Method == method && router.Path == path {
			// 解析请求体
			var request interface{}
			if router.Request != nil {
				// 创建注册类型的新实例
				requestType := reflect.TypeOf(router.Request)
				isPointer := requestType.Kind() == reflect.Ptr
				if isPointer {
					requestType = requestType.Elem()
				}

				// 创建实例指针用于 JSON 绑定
				newValue := reflect.New(requestType)

				if err := c.BindJSON(newValue.Interface()); err != nil {
					return usererrors.New("invalid_request", "Invalid request format")
				}

				// 如果原始类型是指针，返回指针；否则返回值
				if isPointer {
					request = newValue.Interface()
				} else {
					request = newValue.Elem().Interface()
				}
			}

			// 调用处理器
			response, err := router.Handler(c, request)
			if err != nil {
				return err
			}

			return c.Render(response)
		}
	}

	return usererrors.New("endpoint_not_found", "API endpoint not found")
}
