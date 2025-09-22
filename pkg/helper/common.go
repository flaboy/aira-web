package helper

import (
	"strings"

	"github.com/flaboy/aira-web/pkg/config"
	"github.com/flaboy/pin"
)

func BuildUrl(path string) string {
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	if !strings.HasSuffix(config.Config.FrontURL, "/") {
		config.Config.FrontURL += "/"
	}
	return config.Config.FrontURL + path
}

func RemoteIP(c *pin.Context) string {
	// HTTP头一般格式如下:
	// X-Forwarded-For: client1, proxy1, proxy2
	// 一般取X-Forwarded-For中第一个非unknown的有效IP字符串
	// 如果都是unknown则取X-Real-IP
	// 如果都没有则取RemoteAddr
	// return c.Request.RemoteAddr
	ip_headers := []string{
		"CF-Connecting-IP",
		"X-Forwarded-For",
		"X-Real-IP",
	}
	for _, header := range ip_headers {
		ip := c.Request.Header.Get(header)
		// revel.AppLog.Infof("ApiController::RemoteIP: %s: %s", header, ip)
		if ip != "" {
			parts := strings.Split(ip, ",")
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}

	// revel.AppLog.Infof("ApiController::RemoteIP: c.Request.RemoteAddr: %s", c.Request.RemoteAddr)
	parts := strings.Split(c.Request.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return c.Request.RemoteAddr
}
