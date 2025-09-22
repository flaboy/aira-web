# Aira Web - HTTP Handlers

HTTP 处理器文档

## 路由

```go
web.GET("/api/users", getUsersHandler)
web.POST("/api/users", createUserHandler)
```

## 中间件

```go
web.Use(authMiddleware)
web.Use(loggingMiddleware)
```
