package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/flaboy/aira-web/pkg/auth"
)

// FacebookProvider Facebook OAuth提供商
type FacebookProvider struct {
	appID string
}

// FacebookUserDetails Facebook用户详情
type FacebookUserDetails struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

// NewFacebookProvider 创建Facebook提供商
func NewFacebookProvider(appID string) auth.CredentialProvider {
	return &FacebookProvider{appID: appID}
}

func (p *FacebookProvider) Name() auth.ProviderType {
	return auth.ProviderFacebook
}

func (p *FacebookProvider) ValidateCredential(ctx context.Context, credential map[string]string) (*auth.ExternalUserInfo, error) {
	// 获取Facebook Access Token
	accessToken, ok := credential["accessToken"]
	if !ok {
		return nil, fmt.Errorf("missing accessToken field")
	}

	// 调用Facebook Graph API验证Token并获取用户信息
	userDetails, err := p.fetchUserDetails(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Facebook user details: %w", err)
	}

	// 转换为标准用户信息格式
	userInfo := &auth.ExternalUserInfo{
		UID:    userDetails.ID,
		Email:  userDetails.Email,
		Name:   userDetails.Name,
		Avatar: userDetails.Picture.Data.URL,
		Metadata: map[string]interface{}{
			"provider": "facebook",
		},
	}

	return userInfo, nil
}

// fetchUserDetails 从Facebook Graph API获取用户详情
func (p *FacebookProvider) fetchUserDetails(ctx context.Context, accessToken string) (*FacebookUserDetails, error) {
	url := fmt.Sprintf("https://graph.facebook.com/me?fields=id,name,email,picture&access_token=%s", accessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Facebook API returned status %d", resp.StatusCode)
	}

	var userDetails FacebookUserDetails
	if err := json.NewDecoder(resp.Body).Decode(&userDetails); err != nil {
		return nil, err
	}

	return &userDetails, nil
}

func (p *FacebookProvider) GetFrontendConfig() *auth.ProviderFrontendConfig {
	return &auth.ProviderFrontendConfig{
		Name:        "Facebook",
		Description: "Facebook signin",
		ConfigJSON:  fmt.Sprintf(`{"client_id":"%s"}`, p.appID),
		LogoutScript: `
			return new Promise( 
				function(resolve, reject){
					let js = document.createElement('script');
					js.src = "https://connect.facebook.net/en_US/sdk.js";
					js.onload = function() {
						FB.init({
							appId      : '',
							cookie     : true,
							xfbml      : true,
							version    : 'v1.0'
						});
						FB.getLoginStatus(function(rsp){
							if(rsp=='connect'){
								FB.logout(function(response){
									resolve(response);
								});
							}else{
								resolve(rsp);
							}
						})
					};
					document.body.appendChild(js)
				}
			)`,
	}
}
