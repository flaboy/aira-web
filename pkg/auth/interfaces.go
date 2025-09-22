package auth

import (
	"context"
	"time"
)

// 🚀 第三方认证服务（工具化设计，支持泛型context）
type ThirdPartyAuthService[TContext any] interface {
	// 🎯 核心方法：用户认证和管理（context由业务层定义）
	AuthenticateUser(ctx context.Context, request *ThirdPartyAuthRequest[TContext]) (*ThirdPartyAuthResult, error)

	// 获取支持的第三方认证方法
	GetAuthMethods() (*AuthMethodsResponse, error)
}

// 🚀 第三方认证请求（泛型context）
type ThirdPartyAuthRequest[TContext any] struct {
	Provider   string                 `json:"provider"`   // "google", "facebook"
	Credential map[string]string      `json:"credential"` // provider特定的凭证
	Context    TContext               `json:"-"`          // 业务层自定义context（可能包含事务，也可能不包含）
	Options    *AuthOptions[TContext] `json:"options"`    // 认证选项
}

// 🚀 认证选项（业务定制点，使用泛型context）
type AuthOptions[TContext any] struct {
	AutoCreateUser bool `json:"auto_create_user"` // 是否自动创建用户

	// 🔗 钩子函数：context由业务层定义（可能包含事务，也可能不包含）
	UserCreationHook   func(ctx TContext, externalInfo *ExternalUserInfo) (interface{}, error) `json:"-"` // 用户创建钩子
	AccountLinkingHook func(ctx TContext, externalInfo *ExternalUserInfo) (interface{}, error) `json:"-"` // 账号关联钩子
	PostAuthHook       func(ctx TContext, user interface{}, authInfo *AuthInfo) error          `json:"-"` // 认证后钩子
}

// 🚀 第三方认证结果（不包含token）
type ThirdPartyAuthResult struct {
	Success      bool                   `json:"success"`
	User         interface{}            `json:"user"` // 返回具体的User类型，使用时需类型断言
	IsNewUser    bool                   `json:"is_new_user"`
	ExternalInfo *ExternalUserInfo      `json:"external_info"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// 🚀 认证信息（传递给PostAuthHook）
type AuthInfo struct {
	Provider    string    `json:"provider"`
	ExternalUID string    `json:"external_uid"`
	IP          string    `json:"ip"`
	UserAgent   string    `json:"user_agent"`
	Timestamp   time.Time `json:"timestamp"`
	IsNewUser   bool      `json:"is_new_user"`
}

// 🚀 第三方认证服务配置（简化版，工具化）
type ThirdPartyAuthConfig struct {
	// 第三方凭证提供商
	CredentialProviders []CredentialProvider
}

// 🚀 第三方凭证提供商接口
type CredentialProvider interface {
	Name() ProviderType
	ValidateCredential(ctx context.Context, credential map[string]string) (*ExternalUserInfo, error)
	GetFrontendConfig() *ProviderFrontendConfig
}

// 🚀 第三方认证仓库接口（工具化，由业务层实现）
type ThirdPartyAuthRepository[TContext any] interface {
	// 绑定关系管理
	FindBinding(ctx TContext, provider, externalUID string) (*ThirdPartyBinding, error)
	CreateBinding(ctx TContext, userID uint, provider, externalUID string) error
	DeleteBinding(ctx TContext, userID uint, provider string) error
	ListUserBindings(ctx TContext, userID uint) ([]*ThirdPartyBinding, error)

	// 用户信息获取（只读）
	GetUserByID(ctx TContext, userID uint) (interface{}, error) // 返回具体的User类型
}

// 强类型定义
type ProviderType string

// 提供商类型常量
const (
	ProviderGoogle   ProviderType = "google"
	ProviderFacebook ProviderType = "facebook"
	ProviderWeChat   ProviderType = "wechat"
)

// UserWithID 用户ID接口 - 所有用户对象都应该实现此接口
type UserWithID interface {
	GetID() uint
}
