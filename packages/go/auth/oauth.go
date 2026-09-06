// Package auth - OAuth2/OIDC client cho Google, Facebook, Microsoft.
//
// Mỗi provider có struct riêng kế thừa OAuthClient chung, dùng go-oidc cho
// OpenID Connect Discovery và golang.org/x/oauth2 cho authorization code flow.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OAuthProvider định danh provider.
type OAuthProvider string

const (
	ProviderGoogle    OAuthProvider = "google"
	ProviderFacebook  OAuthProvider = "facebook"
	ProviderMicrosoft OAuthProvider = "microsoft"
	ProviderApple     OAuthProvider = "apple"
	ProviderLine      OAuthProvider = "line"
)

// OAuthUserInfo chuẩn hoá thông tin user từ các provider.
type OAuthUserInfo struct {
	Provider       OAuthProvider `json:"provider"`
	ProviderUserID string        `json:"provider_user_id"`
	Email          string        `json:"email"`
	EmailVerified  bool          `json:"email_verified"`
	FullName       string        `json:"full_name"`
	FirstName      string        `json:"first_name"`
	LastName       string        `json:"last_name"`
	AvatarURL      string        `json:"avatar_url"`
	Locale         string        `json:"locale"`
	Raw            json.RawMessage `json:"raw"`
}

// OAuthConfig cấu hình chung cho 1 provider.
type OAuthConfig struct {
	Provider     OAuthProvider
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	ExtraParams  map[string]string
	DiscoveryURL string // cho OIDC provider (Google, Microsoft)
	GraphVersion string // cho Facebook ("v18.0")
}

// OAuthClient là wrapper cho 1 provider.
type OAuthClient struct {
	cfg     OAuthConfig
	oauth2c *oauth2.Config
	verifier *oidc.IDTokenVerifier
	graphBase string
	httpClient *http.Client
}

// NewOAuthClient khởi tạo client cho provider. Nếu provider là OIDC
// (Google/Microsoft), sẽ discovery endpoints tự động.
func NewOAuthClient(ctx context.Context, cfg OAuthConfig) (*OAuthClient, error) {
	c := &OAuthClient{
		cfg: cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	switch cfg.Provider {
	case ProviderGoogle, ProviderMicrosoft, ProviderApple:
		// OIDC providers
		if cfg.DiscoveryURL == "" {
			return nil, fmt.Errorf("oidc: discovery URL required for %s", cfg.Provider)
		}
		provider, err := oidc.NewProvider(ctx, cfg.DiscoveryURL)
		if err != nil {
			return nil, fmt.Errorf("oidc discovery: %w", err)
		}
		endpoint := provider.Endpoint()
		c.oauth2c = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     endpoint,
		}
		c.verifier = provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	case ProviderFacebook:
		// Facebook Graph API - manual endpoint
		graphVer := cfg.GraphVersion
		if graphVer == "" {
			graphVer = "v18.0"
		}
		c.graphBase = "https://graph.facebook.com/" + graphVer
		c.oauth2c = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.facebook.com/v18.0/dialog/oauth",
				TokenURL: c.graphBase + "/oauth/access_token",
			},
		}
	default:
		return nil, fmt.Errorf("oauth: unknown provider %q", cfg.Provider)
	}

	if len(cfg.Scopes) == 0 {
		c.oauth2c.Scopes = c.defaultScopes()
	}
	return c, nil
}

func (c *OAuthClient) defaultScopes() []string {
	switch c.cfg.Provider {
	case ProviderGoogle, ProviderMicrosoft:
		return []string{"openid", "email", "profile"}
	case ProviderFacebook:
		return []string{"email", "public_profile"}
	case ProviderApple:
		return []string{"name", "email"}
	default:
		return []string{"openid", "email", "profile"}
	}
}

// AuthCodeURL trả về URL để redirect user sang provider.
// state nên là giá trị random lưu ở session để chống CSRF.
func (c *OAuthClient) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	options := []oauth2.AuthCodeOption{}
	for k, v := range c.cfg.ExtraParams {
		options = append(options, oauth2.SetAuthURLParam(k, v))
	}
	options = append(options, opts...)
	return c.oauth2c.AuthCodeURL(state, options...)
}

// Exchange đổi authorization code lấy access_token + id_token (OIDC).
func (c *OAuthClient) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return c.oauth2c.Exchange(ctx, code, opts...)
}

// FetchUserInfo lấy thông tin user từ provider.
func (c *OAuthClient) FetchUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	switch c.cfg.Provider {
	case ProviderGoogle, ProviderMicrosoft, ProviderApple:
		return c.fetchOIDCUserInfo(ctx, token)
	case ProviderFacebook:
		return c.fetchFacebookUserInfo(ctx, token)
	default:
		return nil, fmt.Errorf("oauth: FetchUserInfo not implemented for %s", c.cfg.Provider)
	}
}

func (c *OAuthClient) fetchOIDCUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("oauth: id_token missing in token response")
	}
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oauth: verify id_token: %w", err)
	}
	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Locale        string `json:"locale"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("oauth: parse id_token claims: %w", err)
	}
	raw, _ := json.Marshal(claims)
	return &OAuthUserInfo{
		Provider:       c.cfg.Provider,
		ProviderUserID: claims.Sub,
		Email:          claims.Email,
		EmailVerified:  claims.EmailVerified,
		FullName:       claims.Name,
		FirstName:      claims.GivenName,
		LastName:       claims.FamilyName,
		AvatarURL:      claims.Picture,
		Locale:         claims.Locale,
		Raw:            raw,
	}, nil
}

func (c *OAuthClient) fetchFacebookUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	// Facebook: token endpoint trả access_token + token_type. Gọi /me với fields.
	client := c.oauth2c.Client(ctx, token)
	u := c.graphBase + "/me?fields=id,name,first_name,last_name,email,picture.type(large),locale"
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("facebook /me: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("facebook /me status=%d: %s", resp.StatusCode, string(body))
	}
	var data struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Picture   struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
		Locale string `json:"locale"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("facebook decode: %w", err)
	}
	raw, _ := json.Marshal(data)
	return &OAuthUserInfo{
		Provider:       ProviderFacebook,
		ProviderUserID: data.ID,
		Email:          data.Email,
		EmailVerified:  data.Email != "", // Facebook always returns email if verified
		FullName:       data.Name,
		FirstName:      data.FirstName,
		LastName:       data.LastName,
		AvatarURL:      data.Picture.Data.URL,
		Locale:         data.Locale,
		Raw:            raw,
	}, nil
}

// ===== State helpers =====

// GenerateState tạo CSRF state ngẫu nhiên.
func GenerateState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidateState kiểm tra state.
func ValidateState(expected, got string) bool {
	return expected != "" && got != "" && expected == got
}

// ===== PKCE helpers =====

// GeneratePKCE tạo code_verifier và code_challenge cho S256 PKCE flow.
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	// S256: challenge = base64url(sha256(verifier))
	h := sha256Sum([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h)
	return
}

// sha256Sum dùng SHA-256 không import trực tiếp để tránh naming clash.
func sha256Sum(data []byte) []byte {
	// Local import để tránh đưa crypto/sha256 vào public API
	return sha256sumImpl(data)
}

// ===== Token refresh =====

// RefreshAccessToken dùng refresh_token để lấy access_token mới.
func (c *OAuthClient) RefreshAccessToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	src := c.oauth2c.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	return src.Token()
}

// Client trả về HTTP client dùng access_token của user.
func (c *OAuthClient) Client(ctx context.Context, t *oauth2.Token) *http.Client {
	return c.oauth2c.Client(ctx, t)
}

// EncodeStateToCookie serialize state + PKCE + return_to thành 1 string cookie.
// Production nên mã hoá/cookie-sign, đây là minimal helper.
func EncodeStateToCookie(parts map[string]string) string {
	v := url.Values{}
	for k, v2 := range parts {
		v.Set(k, v2)
	}
	return v.Encode()
}

// DecodeStateFromCookie ngược lại.
func DecodeStateFromCookie(raw string) (map[string]string, error) {
	out := make(map[string]string)
	for _, kv := range strings.Split(raw, "&") {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			continue
		}
		out[pair[0]] = pair[1]
	}
	if len(out) == 0 {
		return nil, errors.New("oauth: empty state cookie")
	}
	return out, nil
}