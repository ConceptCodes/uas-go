package helpers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"uas/config"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/scrypt"
)

type EncryptionHelper struct {
	log *zerolog.Logger
	key []byte
}

func NewEncryptionHelper(log *zerolog.Logger) (*EncryptionHelper, error) {
	key := []byte(config.AppConfig.EncryptionKey)
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}

	return &EncryptionHelper{
		log: log,
		key: key,
	}, nil
}

func (e *EncryptionHelper) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nil, []byte(plaintext), nil)

	result := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

func (e *EncryptionHelper) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encrypted := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func (e *EncryptionHelper) DeriveKey(password, salt string) ([]byte, error) {
	return scrypt.Key([]byte(password), []byte(salt), 32768, 8, 1, 32)
}

func (e *EncryptionHelper) GenerateSalt() (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

func (e *EncryptionHelper) MaskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		if len(email) <= 2 {
			return "***"
		}
		return email[:1] + "***" + email[len(email)-1:]
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 1 {
		return "***" + domain
	}
	return local[:1] + "***" + local[len(local)-1:] + domain
}

func (e *EncryptionHelper) MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:2] + "****" + phone[len(phone)-2:]
}

func (e *EncryptionHelper) MaskName(name string) string {
	if len(name) <= 2 {
		return name[:1] + "*"
	}
	return name[:1] + strings.Repeat("*", len(name)-1)
}
