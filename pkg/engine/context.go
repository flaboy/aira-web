package engine

import (
	"github.com/flaboy/pin"
)

type Context struct {
	*pin.Context
	// 扩展字段
	UserID    uint
	TenantID  uint
	RequestID string
}

func NewContext(c *pin.Context) *Context {
	return &Context{
		Context: c,
	}
}
