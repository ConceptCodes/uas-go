package helpers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
	"uas/config"
	"uas/internal/constants"
	repository "uas/internal/repositories"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

func GenerateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}

type AuthHelper struct {
	log            *zerolog.Logger
	departmentRepo repository.DepartmentRepository
	redisHelper    RedisHelper
	tokenHelper    *TokenHelper
}

func NewAuthHelper(log *zerolog.Logger, departmentRepo repository.DepartmentRepository, redisHelper RedisHelper) *AuthHelper {
	return &AuthHelper{log: log, departmentRepo: departmentRepo, redisHelper: redisHelper}
}

func (h *AuthHelper) WithTokenHelper(th *TokenHelper) *AuthHelper {
	h.tokenHelper = th
	return h
}

func (h *AuthHelper) GenerateBasicAuthToken(tenantId string, tenantSecret string) string {
	h.log.Debug().Msgf("Generating basic auth token for tenant: %s", tenantId)
	return base64.StdEncoding.EncodeToString([]byte(tenantId + ":" + tenantSecret))
}

func (h *AuthHelper) ValidateBasicAuthToken(token string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(token)

	if err != nil {
		h.log.Error().Err(err).Msg("Error decoding token")
		return "", err
	}

	parts := strings.Split(string(data), ":")
	if len(parts) < 2 {
		return "", errors.New("invalid token format")
	}

	tenantId := parts[0]
	tenantSecret := parts[1]

	tenant, err := h.departmentRepo.FindById(tenantId)

	if err != nil {
		return "", err
	}

	valid := h.CheckPasswordHash(tenantSecret, tenant.Secret)

	if valid {
		return tenantId, nil
	}

	return "", errors.New("invalid tenant credentials")
}

func (h *AuthHelper) HashPassword(password string) (string, error) {
	h.log.Debug().Msg("Hashing password")
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (h *AuthHelper) CheckPasswordHash(password, hash string) bool {
	h.log.Debug().Msg("Checking password hash")
	if password == "" || hash == "" {
		h.log.Error().Msg("Password or hash is empty")
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (h *AuthHelper) GenerateAuthToken() string {
	a := uuid.New().String()
	b := uuid.New().String()

	return h.GenerateBasicAuthToken(a, b)
}

func (h *AuthHelper) GenerateOtpCode(target string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("failed to generate secure OTP: %w", err)
	}
	otpCode := fmt.Sprintf("%06d", n.Int64())

	key := fmt.Sprintf("otp:%s", target)
	dur := time.Duration(config.AppConfig.OtpExpire) * time.Minute

	err = h.redisHelper.SetData(key, otpCode, dur)

	if err != nil {
		h.log.Error().Err(err).Msg("Error generating OTP code")
		return "", err
	}

	return otpCode, nil
}

func (h *AuthHelper) ValidateOtpCode(target string, otpCode string) error {
	key := fmt.Sprintf("otp:%s", target)

	code, err := h.redisHelper.GetData(key)

	if err != nil {
		h.log.Error().Err(err).Msg("Error getting OTP code")
		return err
	}

	if code == "" {
		return errors.New("otp code not found")
	}

	if subtle.ConstantTimeCompare([]byte(otpCode), []byte(code)) != 1 {
		return errors.New("invalid OTP code")
	}

	if err := h.redisHelper.DeleteData(key); err != nil {
		h.log.Warn().Err(err).Msg("Failed to delete OTP after successful verification")
	}

	return nil
}

func (h *AuthHelper) GenerateAccessCookie(access_token string, w http.ResponseWriter) {
	if access_token == "" {
		return
	}

	cookieHashKey := []byte(config.AppConfig.CookieHashKey)
	cookieBlockKey := []byte(config.AppConfig.CookieBlockKey)

	var s = securecookie.New(cookieHashKey, cookieBlockKey)

	if encoded, err := s.Encode(constants.AccessTokenCookie, access_token); err == nil {
		sameSite := http.SameSiteStrictMode
		if config.AppConfig.CookieSameSite == "Lax" {
			sameSite = http.SameSiteLaxMode
		}

		cookie := &http.Cookie{
			Name:     constants.AccessTokenCookie,
			Value:    encoded,
			Path:     "/",
			Domain:   config.AppConfig.CookieDomain,
			MaxAge:   config.AppConfig.AccessJwtExpire * 3600,
			Secure:   config.AppConfig.CookieSecure,
			HttpOnly: true,
			SameSite: sameSite,
		}
		http.SetCookie(w, cookie)
	}
}

func (h *AuthHelper) DecodeAccessCookie(encoded string) (string, error) {
	if encoded == "" {
		return "", errors.New("empty access token cookie")
	}

	cookieHashKey := []byte(config.AppConfig.CookieHashKey)
	cookieBlockKey := []byte(config.AppConfig.CookieBlockKey)
	sc := securecookie.New(cookieHashKey, cookieBlockKey)

	var token string
	if err := sc.Decode(constants.AccessTokenCookie, encoded, &token); err != nil {
		return "", err
	}

	return token, nil
}
