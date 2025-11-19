package models

import (
	"context"
	"database/sql/driver"
	"fmt"
	"os"

	"uas/internal/helpers"

	"github.com/rs/zerolog"
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

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	encryptionHelper, err := helpers.NewEncryptionHelper(&log)
	if err != nil {
		return fmt.Errorf("failed to create encryption helper: %w", err)
	}

	decrypted, err := encryptionHelper.Decrypt(encryptedStr)
	if err != nil {
		return fmt.Errorf("failed to decrypt field: %w", err)
	}

	ef.Data = decrypted
	return nil
}

func (ef EncryptedField) Value() (driver.Value, error) {
	if ef.Data == "" {
		return "", nil
	}

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	encryptionHelper, err := helpers.NewEncryptionHelper(&log)
	if err != nil {
		return "", fmt.Errorf("failed to create encryption helper: %w", err)
	}

	encrypted, err := encryptionHelper.Encrypt(ef.Data)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt field: %w", err)
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

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	encryptionHelper, err := helpers.NewEncryptionHelper(&log)
	if err != nil {
		return clause.Expr{SQL: "NULL"}
	}

	encrypted, err := encryptionHelper.Encrypt(ef.Data)
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
