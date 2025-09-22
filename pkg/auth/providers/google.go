package providers

import (
	"context"
	"fmt"

	"github.com/flaboy/aira-web/pkg/auth"
	"google.golang.org/api/idtoken"
)

// GoogleProvider Google OAuth提供商
type GoogleProvider struct {
	clientID string
}

// NewGoogleProvider 创建Google提供商
func NewGoogleProvider(clientID string) auth.CredentialProvider {
	return &GoogleProvider{clientID: clientID}
}

func (p *GoogleProvider) Name() auth.ProviderType {
	return auth.ProviderGoogle
}

func (p *GoogleProvider) ValidateCredential(ctx context.Context, credential map[string]string) (*auth.ExternalUserInfo, error) {
	// 获取Google ID Token
	idToken, ok := credential["credential"]
	if !ok {
		return nil, fmt.Errorf("missing credential field")
	}

	// 验证Token
	payload, err := idtoken.Validate(ctx, idToken, p.clientID)
	if err != nil {
		return nil, fmt.Errorf("invalid Google token: %w", err)
	}

	// 验证发行者
	if payload.Issuer != "https://accounts.google.com" {
		return nil, fmt.Errorf("invalid issuer: %v", payload.Issuer)
	}

	// 验证audience
	if payload.Audience != p.clientID {
		return nil, fmt.Errorf("invalid audience: %v", payload.Audience)
	}

	// 提取用户信息
	userInfo := &auth.ExternalUserInfo{
		UID:    getString(payload.Claims, "email"),
		Name:   getString(payload.Claims, "name"),
		Avatar: getString(payload.Claims, "picture"),
		Locale: getString(payload.Claims, "locale"),
		Metadata: map[string]interface{}{
			"given_name":  getString(payload.Claims, "given_name"),
			"family_name": getString(payload.Claims, "family_name"),
		},
	}

	// 只有验证过的邮箱才设置
	if payload.Claims["email_verified"] == true {
		userInfo.Email = getString(payload.Claims, "email")
	}

	return userInfo, nil
}

func (p *GoogleProvider) GetFrontendConfig() *auth.ProviderFrontendConfig {
	return &auth.ProviderFrontendConfig{
		Name:        "Google",
		Description: "Google one tap signin",
		ConfigJSON:  fmt.Sprintf(`{"client_id":"%s"}`, p.clientID),
	}
}

// getString 安全地从claims中获取字符串值
func getString(claims map[string]interface{}, key string) string {
	if value, ok := claims[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
