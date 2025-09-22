package errors

import "github.com/flaboy/pin/usererrors"

// Web控制器相关错误
var (
	ErrInvalidURL             = usererrors.New("web.invalid_url", "Invalid URL")
	ErrBusinessContextMarshal = usererrors.New("web.business_context_marshal_failed", "Failed to marshal business context")
	ErrBusinessShopCreation   = usererrors.New("web.business_shop_creation_failed", "Failed to create business shop")
	ErrPlatformNotSupported   = usererrors.New("web.platform_not_supported", "Unsupported platform")
)
