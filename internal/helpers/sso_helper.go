package helpers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
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
	if provider.JWKSURI == "" {
		return nil, errors.New("ID token verification requires JWKSURI on the provider")
	}

	keys, err := h.fetchJWKS(provider.JWKSURI)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	parsed, err := jwt.Parse(idToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		for _, k := range keys {
			if kid == "" || k.kid == kid {
				return k.key, nil
			}
		}
		return nil, errors.New("no matching key found in JWKS")
	}, jwt.WithAudience(provider.ClientID), jwt.WithIssuer(provider.IssuerURL))

	if err != nil {
		return nil, fmt.Errorf("ID token verification failed: %w", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid ID token claims")
	}

	return claims, nil
}

type jwksKey struct {
	kid string
	key interface{}
}

func (h *SSOHelper) fetchJWKS(jwksURI string) ([]jwksKey, error) {
	req, err := http.NewRequest("GET", jwksURI, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return nil, err
	}

	var jwks struct {
		Keys []struct {
			Kty string   `json:"kty"`
			Kid string   `json:"kid"`
			Use string   `json:"use"`
			N   string   `json:"n"`
			E   string   `json:"e"`
			X   string   `json:"x"`
			Y   string   `json:"y"`
			Crv string   `json:"crv"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}

	var keys []jwksKey
	for _, k := range jwks.Keys {
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		switch k.Kty {
		case "RSA":
			nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
			if err != nil {
				continue
			}
			eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
			if err != nil {
				continue
			}
			ei := 0
			for _, b := range eBytes {
				ei = ei<<8 + int(b)
			}
			pub := &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: ei}
			keys = append(keys, jwksKey{kid: k.Kid, key: pub})
		case "EC":
			xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
			if err != nil {
				continue
			}
			yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
			if err != nil {
				continue
			}
			var curve elliptic.Curve
			switch k.Crv {
			case "P-256":
				curve = elliptic.P256()
			case "P-384":
				curve = elliptic.P384()
			case "P-521":
				curve = elliptic.P521()
			default:
				continue
			}
			pub := &ecdsa.PublicKey{Curve: curve, X: new(big.Int).SetBytes(xBytes), Y: new(big.Int).SetBytes(yBytes)}
			keys = append(keys, jwksKey{kid: k.Kid, key: pub})
		}
	}

	if len(keys) == 0 {
		return nil, errors.New("no usable keys found in JWKS")
	}
	return keys, nil
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

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub token response: %w", err)
	}
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