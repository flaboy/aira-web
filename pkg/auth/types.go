package auth

import (
	"time"
)

// Entity 通用实体结构
type Entity struct {
	ID        uint                   `json:"id"`
	Email     string                 `json:"email,omitempty"`
	Phone     string                 `json:"phone,omitempty"`
	Username  string                 `json:"username,omitempty"`
	Name      string                 `json:"name"`
	Avatar    string                 `json:"avatar,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // 扩展字段
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ExternalUserInfo 第三方用户信息
type ExternalUserInfo struct {
	UID      string                 `json:"uid"`
	Email    string                 `json:"email,omitempty"`
	Phone    string                 `json:"phone,omitempty"`
	Username string                 `json:"username,omitempty"`
	Name     string                 `json:"name"`
	Avatar   string                 `json:"avatar,omitempty"`
	Locale   string                 `json:"locale,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LoginResult 登录结果
type LoginResult struct {
	Token             string  `json:"token"`
	Entity            *Entity `json:"entity"`
	RequiresTwoFactor bool    `json:"requires_two_factor"`
	ProcessID         string  `json:"process_id,omitempty"`
}

// AuthError 认证错误
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// ProviderFrontendConfig 提供商前端配置（简化版 - 约定大于配置）
type ProviderFrontendConfig struct {
	Name         string `json:"name"`                   // 显示名称
	Description  string `json:"description"`            // 描述
	ConfigJSON   string `json:"configJson"`             // 客户端配置JSON
	LogoutScript string `json:"logoutScript,omitempty"` // 登出脚本（可选）
}

// AuthMethodsResponse 认证方法响应（兼容前端格式）
type AuthMethodsResponse struct {
	AuthMethods []AuthMethodInfo `json:"auth_methods"`
}

// AuthMethodInfo 单个认证方法信息（约定大于配置）
type AuthMethodInfo struct {
	UUID         string            `json:"UUID"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Component    map[string]string `json:"component"`
	ConfigJSON   string            `json:"configJson"`
	LogoutScript string            `json:"logoutScript,omitempty"`
}

// 🚀 新增：第三方绑定信息
type ThirdPartyBinding struct {
	Provider    string    `json:"provider"`
	ExternalUID string    `json:"external_uid"`
	UserID      uint      `json:"user_id"`
	BoundAt     time.Time `json:"bound_at"`
	IsActive    bool      `json:"is_active"`
}

// 预定义错误
var (
	ErrInvalidCredentials     = &AuthError{Code: "invalid_credentials", Message: "Invalid credentials"}
	ErrAccountNotFound        = &AuthError{Code: "account_not_found", Message: "Account not found"}
	ErrProviderNotSupported   = &AuthError{Code: "provider_not_supported", Message: "Provider not supported"}
	ErrIdentifierNotSupported = &AuthError{Code: "identifier_not_supported", Message: "Identifier type not supported"}
	ErrInvalidToken           = &AuthError{Code: "invalid_token", Message: "Invalid token"}
	ErrAccountAlreadyLinked   = &AuthError{Code: "account_already_linked", Message: "Account already linked to another user"}
)
