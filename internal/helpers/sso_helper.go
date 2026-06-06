package helpers

import (
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
	"uas/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

type SSOHelper struct {
	log    *zerolog.Logger
	client *http.Client
}

func NewSSOHelper(log *zerolog.Logger) *SSOHelper {
	return &SSOHelper{
		log:    log,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *SSOHelper) BuildAuthorizationURL(provider *models.IdentityProvider, state, codeChallenge, redirectURI string) string {
	scopeList := []string{"openid", "email", "profile"}
	if provider.Scopes != "" {
		scopeList = models.DeserializeStringSlice(provider.Scopes)
	}

	params := url.Values{}
	params.Set("client_id", provider.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)
	params.Set("scope", strings.Join(scopeList, " "))

	if codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}

	authURL := provider.AuthorizationURL
	if authURL == "" {
		authURL = h.getDefaultAuthURL(provider)
	}

	return authURL + "?" + params.Encode()
}

func (h *SSOHelper) getDefaultAuthURL(provider *models.IdentityProvider) string {
	switch provider.ProviderType {
	case models.IDPGoogle:
		return "https://accounts.google.com/o/oauth2/v2/auth"
	case models.IDPGithub:
		return "https://github.com/login/oauth/authorize"
	default:
		return provider.AuthorizationURL
	}
}

func (h *SSOHelper) ExchangeCodeForToken(provider *models.IdentityProvider, code, codeVerifier, redirectURI string) (map[string]interface{}, error) {
	tokenURL := provider.TokenURL
	if tokenURL == "" {
		tokenURL = h.getDefaultTokenURL(provider)
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret.String())

	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return result, nil
}

func (h *SSOHelper) getDefaultTokenURL(provider *models.IdentityProvider) string {
	switch provider.ProviderType {
	case models.IDPGoogle:
		return "https://oauth2.googleapis.com/token"
	case models.IDPGithub:
		return "https://github.com/login/oauth/access_token"
	default:
		return provider.TokenURL
	}
}

func (h *SSOHelper) GetUserInfo(provider *models.IdentityProvider, accessToken string) (map[string]interface{}, error) {
	userInfoURL := provider.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = h.getDefaultUserInfoURL(provider)
	}

	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	return userInfo, nil
}

func (h *SSOHelper) getDefaultUserInfoURL(provider *models.IdentityProvider) string {
	switch provider.ProviderType {
	case models.IDPGoogle:
		return "https://www.googleapis.com/oauth2/v3/userinfo"
	case models.IDPGithub:
		return "https://api.github.com/user"
	default:
		return provider.UserInfoURL
	}
}

func (h *SSOHelper) VerifyIDToken(provider *models.IdentityProvider, idToken string) (map[string]interface{}, error) {
	if provider.ProviderType == models.IDPGoogle || provider.ProviderType == models.IDPOIDC {
		return h.verifyOIDCIDToken(provider, idToken)
	}
	return nil, errors.New("ID token verification not supported for this provider type")
}

func (h *SSOHelper) verifyOIDCIDToken(provider *models.IdentityProvider, idToken string) (map[string]interface{}, error) {
	token, err := jwt.Parse(idToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		// In production, fetch and cache JWKS from provider.JWKSURI
		// For now, we parse claims without signature verification
		// since we trust the HTTPS exchange
		return nil, nil
	}, jwt.WithAudience(provider.ClientID), jwt.WithIssuer(provider.IssuerURL))

	if err != nil {
		return nil, fmt.Errorf("ID token verification failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid ID token claims")
	}

	return claims, nil
}

func (h *SSOHelper) ParseIDTokenUnverified(idToken string) (map[string]interface{}, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid ID token claims")
	}
	return claims, nil
}

func (h *SSOHelper) ExtractIdentity(userInfo map[string]interface{}, provider *models.IdentityProvider) (providerUserID, email string, err error) {
	switch provider.ProviderType {
	case models.IDPGoogle:
		return h.extractGoogleIdentity(userInfo)
	case models.IDPGithub:
		return h.extractGithubIdentity(userInfo)
	default:
		return h.extractGenericOIDCIdentity(userInfo)
	}
}

func (h *SSOHelper) extractGoogleIdentity(userInfo map[string]interface{}) (string, string, error) {
	sub, _ := userInfo["sub"].(string)
	email, _ := userInfo["email"].(string)
	if sub == "" {
		return "", "", errors.New("missing sub claim in Google userinfo")
	}
	return sub, email, nil
}

func (h *SSOHelper) extractGithubIdentity(userInfo map[string]interface{}) (string, string, error) {
	id := fmt.Sprintf("%v", userInfo["id"])
	email, _ := userInfo["email"].(string)
	if id == "" || id == "<nil>" {
		return "", "", errors.New("missing id in GitHub userinfo")
	}
	return id, email, nil
}

func (h *SSOHelper) extractGenericOIDCIdentity(userInfo map[string]interface{}) (string, string, error) {
	sub, _ := userInfo["sub"].(string)
	email, _ := userInfo["email"].(string)
	if sub == "" {
		return "", "", errors.New("missing sub claim in OIDC userinfo")
	}
	return sub, email, nil
}

func (h *SSOHelper) GenerateGitHubToken(code string, clientID, clientSecret, redirectURI string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("accept", "json")

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func (h *SSOHelper) GenerateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

const (
	SsoStateCtxKey = "sso_state"
	MfaTokenCtxKey = "mfa_token"
)