package auth

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"
)

// thirdPartyAuthService 第三方认证服务实现（工具化）
type thirdPartyAuthService[TContext any] struct {
	config              *ThirdPartyAuthConfig
	credentialProviders map[ProviderType]CredentialProvider
	providerOrder       []ProviderType // 保持注册顺序
	repository          ThirdPartyAuthRepository[TContext]
}

// NewThirdPartyAuthService 创建第三方认证服务（泛型版本）
func NewThirdPartyAuthService[TContext any](config *ThirdPartyAuthConfig, repository ThirdPartyAuthRepository[TContext]) ThirdPartyAuthService[TContext] {
	providers := make(map[ProviderType]CredentialProvider)
	var order []ProviderType

	for _, provider := range config.CredentialProviders {
		providerType := provider.Name()
		providers[providerType] = provider
		order = append(order, providerType)
	}

	return &thirdPartyAuthService[TContext]{
		config:              config,
		credentialProviders: providers,
		providerOrder:       order,
		repository:          repository,
	}
}

// AuthenticateUser 🚀 核心方法：第三方用户认证（工具化，不包含token生成）
func (s *thirdPartyAuthService[TContext]) AuthenticateUser(ctx context.Context, request *ThirdPartyAuthRequest[TContext]) (*ThirdPartyAuthResult, error) {
	// 🔧 步骤1：验证第三方凭证
	externalInfo, err := s.validateCredential(ctx, request.Provider, request.Credential)
	if err != nil {
		return nil, fmt.Errorf("credential validation failed: %w", err)
	}

	// 🔧 步骤2：查找现有绑定关系
	existingBinding, err := s.repository.FindBinding(request.Context, request.Provider, externalInfo.UID)
	if err == nil {
		// 已绑定，直接返回用户
		user, err := s.repository.GetUserByID(request.Context, existingBinding.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get bound user: %w", err)
		}

		// 执行认证后钩子
		if request.Options != nil && request.Options.PostAuthHook != nil {
			authInfo := &AuthInfo{
				Provider:    request.Provider,
				ExternalUID: externalInfo.UID,
				IP:          getClientIP(ctx),
				UserAgent:   getUserAgent(ctx),
				Timestamp:   time.Now(),
				IsNewUser:   false,
			}

			if err := request.Options.PostAuthHook(request.Context, user, authInfo); err != nil {
				return nil, fmt.Errorf("post auth hook failed: %w", err)
			}
		}

		return &ThirdPartyAuthResult{
			Success:      true,
			User:         user,
			IsNewUser:    false,
			ExternalInfo: externalInfo,
		}, nil
	}

	// 🔧 步骤3：处理新用户或关联现有用户
	var user interface{}
	var isNewUser bool

	// 🔗 步骤3a：查找邮箱匹配的用户（使用业务钩子）
	if request.Options != nil && request.Options.AccountLinkingHook != nil {
		foundUser, err := request.Options.AccountLinkingHook(request.Context, externalInfo)
		if err != nil {
			return nil, fmt.Errorf("account linking hook failed: %w", err)
		}
		user = foundUser
	}

	// 👤 步骤3b：如果没有找到用户且允许自动创建
	if user == nil && request.Options != nil && request.Options.AutoCreateUser {
		if request.Options.UserCreationHook != nil {
			createdUser, err := request.Options.UserCreationHook(request.Context, externalInfo)
			if err != nil {
				return nil, fmt.Errorf("user creation hook failed: %w", err)
			}
			user = createdUser
			isNewUser = true
		} else {
			return nil, fmt.Errorf("user creation hook is required when auto create is enabled")
		}
	}

	if user == nil {
		return nil, fmt.Errorf("user not found and auto creation disabled")
	}

	// 获取用户ID（通过类型断言）
	userID, err := s.extractUserID(user)
	if err != nil {
		return nil, fmt.Errorf("failed to extract user ID: %w", err)
	}

	// 🔗 步骤4：创建绑定关系
	err = s.repository.CreateBinding(request.Context, userID, request.Provider, externalInfo.UID)
	if err != nil {
		return nil, fmt.Errorf("failed to create binding: %w", err)
	}

	// 📝 步骤5：执行后置钩子
	if request.Options != nil && request.Options.PostAuthHook != nil {
		authInfo := &AuthInfo{
			Provider:    request.Provider,
			ExternalUID: externalInfo.UID,
			IP:          getClientIP(ctx),
			UserAgent:   getUserAgent(ctx),
			Timestamp:   time.Now(),
			IsNewUser:   isNewUser,
		}

		if err := request.Options.PostAuthHook(request.Context, user, authInfo); err != nil {
			return nil, fmt.Errorf("post auth hook failed: %w", err)
		}
	}

	return &ThirdPartyAuthResult{
		Success:      true,
		User:         user,
		IsNewUser:    isNewUser,
		ExternalInfo: externalInfo,
	}, nil
}

// GetAuthMethods 获取支持的第三方认证方法
func (s *thirdPartyAuthService[TContext]) GetAuthMethods() (*AuthMethodsResponse, error) {
	var authMethods []AuthMethodInfo

	// 按注册顺序返回认证方法
	for _, providerType := range s.providerOrder {
		provider := s.credentialProviders[providerType]
		frontendConfig := provider.GetFrontendConfig()

		authMethod := AuthMethodInfo{
			UUID:        string(providerType), // 直接使用provider名称作为UUID
			Name:        frontendConfig.Name,
			Description: frontendConfig.Description,
			Component: map[string]string{
				"path": fmt.Sprintf("@/components/auth/%s.vue", frontendConfig.Name), // 使用path而不是name
			},
			ConfigJSON:   frontendConfig.ConfigJSON,
			LogoutScript: frontendConfig.LogoutScript,
		}

		authMethods = append(authMethods, authMethod)
	}

	return &AuthMethodsResponse{
		AuthMethods: authMethods,
	}, nil
}

// validateCredential 验证第三方凭证
func (s *thirdPartyAuthService[TContext]) validateCredential(ctx context.Context, provider string, credential map[string]string) (*ExternalUserInfo, error) {
	// 直接使用前端传入的provider名称，不做任何映射
	providerType := ProviderType(provider)
	credProvider, exists := s.credentialProviders[providerType]
	if !exists {
		return nil, fmt.Errorf("provider %s not supported", provider)
	}

	return credProvider.ValidateCredential(ctx, credential)
}

// extractUserID 从用户对象中提取ID
func (s *thirdPartyAuthService[TContext]) extractUserID(user interface{}) (uint, error) {
	// 优先使用UserWithID接口
	if userWithID, ok := user.(UserWithID); ok {
		return userWithID.GetID(), nil
	}

	// Fallback: 使用反射获取ID字段（向后兼容）
	v := reflect.ValueOf(user)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("user must be a struct or pointer to struct, got %T", user)
	}

	// 尝试获取ID字段
	idField := v.FieldByName("ID")
	if !idField.IsValid() {
		return 0, fmt.Errorf("user struct does not have ID field and does not implement UserWithID interface, type: %T", user)
	}

	// 检查ID字段类型并转换为uint
	switch idField.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(idField.Uint()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal := idField.Int()
		if intVal < 0 {
			return 0, fmt.Errorf("ID field cannot be negative: %d", intVal)
		}
		return uint(intVal), nil
	default:
		return 0, fmt.Errorf("ID field must be numeric type, got %s", idField.Kind())
	}
}

// getClientIP 获取客户端IP
func getClientIP(ctx context.Context) string {
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		return req.RemoteAddr
	}
	return "unknown"
}

// getUserAgent 获取用户代理
func getUserAgent(ctx context.Context) string {
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		return req.UserAgent()
	}
	return "unknown"
}
