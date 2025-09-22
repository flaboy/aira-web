package framework

import (
	"github.com/flaboy/aira-web/pkg/config"
)

func Start(cfg *config.FrameworkConfig) error {
	config.Config = cfg
	return nil
}

// 兼容性函数 - 保持向后兼容
func Init(cfg *config.FrameworkConfig) error {
	return Start(cfg)
}
