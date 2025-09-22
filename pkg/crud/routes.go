package crud

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type handleDef struct {
	method   string
	path     string
	handlers []gin.HandlerFunc
}

type staticDef struct {
	path string
	dir  string
}

type Routes struct {
	middlewares []gin.HandlerFunc
	handles     []handleDef
	statics     []staticDef
}

func NewRoutes() *Routes {
	return &Routes{
		handles:     make([]handleDef, 0),
		middlewares: make([]gin.HandlerFunc, 0),
		statics:     make([]staticDef, 0),
	}
}

func (o *Routes) Use(handlers ...gin.HandlerFunc) *Routes {
	o.middlewares = append(o.middlewares, handlers...)
	return o
}

func (o *Routes) Handle(method, path string, handlers ...gin.HandlerFunc) *Routes {
	o.handles = append(o.handles, handleDef{
		method:   method,
		path:     path,
		handlers: handlers,
	})
	return o
}

func (o *Routes) Any(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("Any", path, handlers...)
}

func (o *Routes) GET(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("GET", path, handlers...)
}

func (o *Routes) POST(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("POST", path, handlers...)
}

func (o *Routes) DELETE(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("DELETE", path, handlers...)
}

func (o *Routes) PATCH(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("PATCH", path, handlers...)
}

func (o *Routes) PUT(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("PUT", path, handlers...)
}

func (o *Routes) OPTIONS(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("OPTIONS", path, handlers...)
}

func (o *Routes) HEAD(path string, handlers ...gin.HandlerFunc) *Routes {
	return o.Handle("HEAD", path, handlers...)
}

func (o *Routes) Match(methods []string, path string, handlers ...gin.HandlerFunc) *Routes {
	for _, method := range methods {
		o.Handle(method, path, handlers...)
	}
	return o
}

func (o *Routes) StaticFile(path, file string) *Routes {
	o.statics = append(o.statics, staticDef{
		path: path,
		dir:  file,
	})
	return o
}

func (o *Routes) StaticFileFS(path, file string, fs http.FileSystem) *Routes {
	return o.StaticFile(path, file)
}

func (o *Routes) Static(path, dir string) *Routes {
	o.statics = append(o.statics, staticDef{
		path: path,
		dir:  dir,
	})
	return o
}

func (o *Routes) StaticFS(path string, fs http.FileSystem) *Routes {
	return o.Static(path, "")
}

func (o *Routes) RegisterTo(parent gin.IRoutes) {
	for _, static := range o.statics {
		if static.dir != "" {
			parent.Static(static.path, static.dir)
		}
	}
	for _, handle := range o.handles {
		log.Printf("Registering route %s %s", handle.method, handle.path)
		if handle.method == "Any" {
			parent.Any(handle.path, handle.handlers...)
		} else {
			parent.Handle(handle.method, handle.path, handle.handlers...)
		}
	}
	for _, middleware := range o.middlewares {
		parent.Use(middleware)
	}
}
