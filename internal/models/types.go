package models

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"uas/config"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EncryptedField struct {
	Data string
}

func (ef *EncryptedField) Scan(value interface{}) error {
	if value == nil {
		ef.Data = ""
		return nil
	}

	var encryptedStr string
	switch v := value.(type) {
	case string:
		encryptedStr = v
	case []byte:
		encryptedStr = string(v)
	default:
		return fmt.Errorf("cannot scan %T into EncryptedField", value)
	}

	if encryptedStr == "" {
		ef.Data = ""
		return nil
	}

	decrypted, err := decryptField(encryptedStr)
	if err != nil {
		return err
	}

	ef.Data = decrypted
	return nil
}

func (ef EncryptedField) Value() (driver.Value, error) {
	if ef.Data == "" {
		return "", nil
	}

	encrypted, err := encryptField(ef.Data)
	if err != nil {
		return "", err
	}

	return encrypted, nil
}

func (ef EncryptedField) GormDataType() string {
	return "TEXT"
}

func (ef EncryptedField) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	if ef.Data == "" {
		return clause.Expr{SQL: "NULL"}
	}

	encrypted, err := encryptField(ef.Data)
	if err != nil {
		return clause.Expr{SQL: "NULL"}
	}

	return clause.Expr{
		SQL:  "?",
		Vars: []interface{}{encrypted},
	}
}

func (ef EncryptedField) String() string {
	return ef.Data
}

func (ef *EncryptedField) Set(value string) {
	ef.Data = value
}

func encryptField(plaintext string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
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
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptField(ciphertext string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
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

func getEncryptionKey() ([]byte, error) {
	key := []byte(config.AppConfig.EncryptionKey)
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	return key, nil
}
