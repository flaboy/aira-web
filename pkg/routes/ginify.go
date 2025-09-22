package routes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flaboy/pin"
)

// GinRouter 是一个基于gin的简化路由器，提供类似gin的API但适配pin.Context
type GinRouter struct {
	basePath string
	routes   []RouteHandler
}

// RouteHandler 路由处理器
type RouteHandler struct {
	Method  string
	Path    string
	Handler func(*pin.Context) error
}

// NewGinRouter 创建新的路由器
func NewGinRouter(basePath string) *GinRouter {
	return &GinRouter{
		basePath: strings.TrimSuffix(basePath, "/"),
		routes:   make([]RouteHandler, 0),
	}
}

// GET 注册GET路由
func (r *GinRouter) GET(path string, handler func(*pin.Context) error) {
	r.routes = append(r.routes, RouteHandler{
		Method:  "GET",
		Path:    path,
		Handler: handler,
	})
}

// POST 注册POST路由
func (r *GinRouter) POST(path string, handler func(*pin.Context) error) {
	r.routes = append(r.routes, RouteHandler{
		Method:  "POST",
		Path:    path,
		Handler: handler,
	})
}

// PUT 注册PUT路由
func (r *GinRouter) PUT(path string, handler func(*pin.Context) error) {
	r.routes = append(r.routes, RouteHandler{
		Method:  "PUT",
		Path:    path,
		Handler: handler,
	})
}

// DELETE 注册DELETE路由
func (r *GinRouter) DELETE(path string, handler func(*pin.Context) error) {
	r.routes = append(r.routes, RouteHandler{
		Method:  "DELETE",
		Path:    path,
		Handler: handler,
	})
}

// PATCH 注册PATCH路由
func (r *GinRouter) PATCH(path string, handler func(*pin.Context) error) {
	r.routes = append(r.routes, RouteHandler{
		Method:  "PATCH",
		Path:    path,
		Handler: handler,
	})
}

// HandleRequest 处理请求，类似gin的路由匹配
func (r *GinRouter) HandleRequest(c *pin.Context, method, requestPath string) error {
	// 移除basePath前缀
	if r.basePath != "" && strings.HasPrefix(requestPath, r.basePath) {
		requestPath = strings.TrimPrefix(requestPath, r.basePath)
	}

	// 如果路径为空或只有斜杠，设为根路径
	if requestPath == "" || requestPath == "/" {
		requestPath = "/"
	}

	for _, route := range r.routes {
		if route.Method == method {
			fmt.Printf("[GinRouter] Matching route: %s %s %s\n", method, route.Path, requestPath)
			if match, params := r.matchPath(route.Path, requestPath); match {
				// 设置路径参数到Context
				for key, value := range params {
					c.Set("param_"+key, value)
				}
				return route.Handler(c)
			}
		}
	}

	return errors.New("route not found: " + method + " " + requestPath)
}

// matchPath 路径匹配，支持参数（:param）和通配符（*）
func (r *GinRouter) matchPath(pattern, path string) (bool, map[string]string) {
	params := make(map[string]string)

	// 分割路径
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	// 根路径特殊处理
	if pattern == "/" && path == "/" {
		return true, params
	}

	if pattern == "/" || path == "/" {
		return pattern == path, params
	}

	// 检查通配符模式
	for i, part := range patternParts {
		if strings.HasPrefix(part, "*") {
			// 通配符匹配，收集剩余路径
			if i < len(pathParts) {
				remaining := strings.Join(pathParts[i:], "/")
				paramName := strings.TrimPrefix(part, "*")
				if paramName == "" {
					paramName = "wildcard"
				}
				params[paramName] = remaining
			}
			return true, params
		}

		if i >= len(pathParts) {
			return false, params
		}

		if strings.HasPrefix(part, ":") {
			// 参数匹配
			paramName := strings.TrimPrefix(part, ":")
			params[paramName] = pathParts[i]
		} else if part != pathParts[i] {
			// 字面量不匹配
			return false, params
		}
	}

	// 检查路径长度是否匹配
	return len(patternParts) == len(pathParts), params
}

// GetParam 从context获取路径参数
func GetParam(c *pin.Context, key string) string {
	if value, exists := c.Get("param_" + key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// Group 创建路由组
func (r *GinRouter) Group(prefix string, middleware ...func(*pin.Context) error) *GinRouterGroup {
	return &GinRouterGroup{
		parent:      r,
		prefix:      prefix,
		middlewares: middleware,
	}
}

// GinRouterGroup 路由组
type GinRouterGroup struct {
	parent      *GinRouter
	prefix      string
	middlewares []func(*pin.Context) error
}

// GET 组内GET路由
func (g *GinRouterGroup) GET(path string, handler func(*pin.Context) error) {
	g.parent.GET(g.prefix+path, g.wrapWithMiddleware(handler))
}

// POST 组内POST路由
func (g *GinRouterGroup) POST(path string, handler func(*pin.Context) error) {
	g.parent.POST(g.prefix+path, g.wrapWithMiddleware(handler))
}

// PUT 组内PUT路由
func (g *GinRouterGroup) PUT(path string, handler func(*pin.Context) error) {
	g.parent.PUT(g.prefix+path, g.wrapWithMiddleware(handler))
}

// DELETE 组内DELETE路由
func (g *GinRouterGroup) DELETE(path string, handler func(*pin.Context) error) {
	g.parent.DELETE(g.prefix+path, g.wrapWithMiddleware(handler))
}

// PATCH 组内PATCH路由
func (g *GinRouterGroup) PATCH(path string, handler func(*pin.Context) error) {
	g.parent.PATCH(g.prefix+path, g.wrapWithMiddleware(handler))
}

// wrapWithMiddleware 包装中间件
func (g *GinRouterGroup) wrapWithMiddleware(handler func(*pin.Context) error) func(*pin.Context) error {
	return func(c *pin.Context) error {
		// 执行中间件
		for _, middleware := range g.middlewares {
			if err := middleware(c); err != nil {
				return err
			}
		}
		// 执行处理器
		return handler(c)
	}
}
