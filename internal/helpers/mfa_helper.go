package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"
	"uas/config"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type MfaHelper struct {
	log         *zerolog.Logger
	redisHelper *RedisHelper
}

func NewMfaHelper(log *zerolog.Logger, redisHelper *RedisHelper) *MfaHelper {
	return &MfaHelper{log: log, redisHelper: redisHelper}
}

func (h *MfaHelper) GenerateTOTPSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.AppConfig.MfaIssuer,
		AccountName: "user",
		Period:      30,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	return key.Secret(), nil
}

func (h *MfaHelper) GetTOTPQRCodeURI(secret, accountName string) string {
	keyURL, _ := totp.Generate(totp.GenerateOpts{
		Issuer:      config.AppConfig.MfaIssuer,
		AccountName: accountName,
		Secret:      []byte(secret),
		Period:      30,
	})
	if keyURL != nil {
		return keyURL.URL()
	}
	return ""
}

func (h *MfaHelper) ValidateTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

func (h *MfaHelper) GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10000000000))
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		code := fmt.Sprintf("%010d", n.Int64())
		formatted := code[:5] + "-" + code[5:]
		codes[i] = formatted
	}
	return codes, nil
}

func (h *MfaHelper) HashBackupCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func (h *MfaHelper) VerifyBackupCode(plainCode string, hashedCodesJSON string) (bool, string, error) {
	var hashedCodes []string
	if err := json.Unmarshal([]byte(hashedCodesJSON), &hashedCodes); err != nil {
		return false, "", err
	}

	plainHash := h.HashBackupCode(plainCode)
	for i, hc := range hashedCodes {
		if hc == plainHash {
			remaining := append(hashedCodes[:i], hashedCodes[i+1:]...)
			remainingJSON, _ := json.Marshal(remaining)
			return true, string(remainingJSON), nil
		}
	}
	return false, "", nil
}

func (h *MfaHelper) CreateMFATempToken(userID, departmentID string) (string, error) {
	token := uuid.New().String()
	key := fmt.Sprintf("mfa:temp:%s", token)
	data := map[string]string{
		"user_id":       userID,
		"department_id": departmentID,
	}
	dataJSON, _ := json.Marshal(data)
	dur := time.Duration(config.AppConfig.MfaChallengeExpireMin) * time.Minute

	err := h.redisHelper.SetData(key, string(dataJSON), dur)
	if err != nil {
		return "", fmt.Errorf("failed to store MFA temp token: %w", err)
	}

	return token, nil
}

func (h *MfaHelper) ValidateMFATempToken(token string) (userID, departmentID string, err error) {
	key := fmt.Sprintf("mfa:temp:%s", token)
	data, err := h.redisHelper.GetData(key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", "", errors.New("MFA token expired or invalid")
		}
		return "", "", err
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return "", "", errors.New("invalid MFA token data")
	}

	h.redisHelper.DeleteData(key)
	return parsed["user_id"], parsed["department_id"], nil
}

func (h *MfaHelper) StoreSSOState(state string, data map[string]string, ttl time.Duration) error {
	dataJSON, _ := json.Marshal(data)
	return h.redisHelper.SetData(fmt.Sprintf("sso:state:%s", state), string(dataJSON), ttl)
}

func (h *MfaHelper) ValidateAndConsumeSSOState(state string) (map[string]string, error) {
	key := fmt.Sprintf("sso:state:%s", state)
	data, err := h.redisHelper.GetData(key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("SSO state expired or invalid")
		}
		return nil, err
	}

	h.redisHelper.DeleteData(key)

	var parsed map[string]string
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, errors.New("invalid SSO state data")
	}

	return parsed, nil
}

func (h *MfaHelper) GenerateNonce() (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(nonce), nil
}

func (h *MfaHelper) GenerateCodeVerifier() (string, string, error) {
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return "", "", fmt.Errorf("failed to generate code verifier: %w", err)
	}
	verifierStr := base64.RawURLEncoding.EncodeToString(verifier)

	sum := sha256.Sum256([]byte(verifierStr))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	return verifierStr, challenge, nil
}

func (h *MfaHelper) AmrClaimForFactor(factorType string) string {
	switch factorType {
	case "totp":
		return "otp"
	case "sms":
		return "sms"
	case "email":
		return "email"
	default:
		return "mfa"
	}
}

const MfaTokenContextKey = "mfa_token"