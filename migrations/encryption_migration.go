package main

import (
	"uas/config"
	"uas/internal/helpers"
	"uas/pkg/logger"
	"uas/pkg/storage/mysql"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// Migration to add encrypted fields to the users table
func main() {
	log := logger.New()

	// Connect to database
	db, err := mysql.New(*log)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while connecting to database")
	}

	// Run migration
	if err := migrateEncryptionFields(db, log); err != nil {
		log.Fatal().Err(err).Msg("Error while running migration")
	}

	log.Info().Msg("Encryption migration completed successfully")
}

func migrateEncryptionFields(db *gorm.DB, log *zerolog.Logger) error {
	// Add encrypted fields to users table
	migrations := []string{
		"ALTER TABLE user_models ADD COLUMN IF NOT EXISTS encrypted_name TEXT;",
		"ALTER TABLE user_models ADD COLUMN IF NOT EXISTS encrypted_email TEXT;",
		"ALTER TABLE user_models ADD COLUMN IF NOT EXISTS encrypted_phone_number TEXT;",
	}

	for _, migration := range migrations {
		if err := db.Exec(migration).Error; err != nil {
			// Check if column already exists
			if !isColumnExistsError(err) {
				return err
			}
		}
	}

	// Encrypt existing data if encryption is enabled
	if config.AppConfig.EnableDBEncryption {
		if err := encryptExistingData(db, log); err != nil {
			return err
		}
	}

	return nil
}

func encryptExistingData(db *gorm.DB, log *zerolog.Logger) error {
	// Get encryption helper
	encryptionHelper, err := helpers.NewEncryptionHelper(log)
	if err != nil {
		return err
	}

	// Fetch all users
	type User struct {
		ID          uint
		Name        string
		Email       string
		PhoneNumber string
	}

	var users []User
	if err := db.Table("user_models").Find(&users).Error; err != nil {
		return err
	}

	// Encrypt and update each user
	for _, user := range users {
		updates := make(map[string]interface{})

		if user.Name != "" {
			encryptedName, err := encryptionHelper.Encrypt(user.Name)
			if err != nil {
				log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to encrypt name")
				continue
			}
			updates["encrypted_name"] = encryptedName
		}

		if user.Email != "" {
			encryptedEmail, err := encryptionHelper.Encrypt(user.Email)
			if err != nil {
				log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to encrypt email")
				continue
			}
			updates["encrypted_email"] = encryptedEmail
		}

		if user.PhoneNumber != "" {
			encryptedPhone, err := encryptionHelper.Encrypt(user.PhoneNumber)
			if err != nil {
				log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to encrypt phone number")
				continue
			}
			updates["encrypted_phone_number"] = encryptedPhone
		}

		if len(updates) > 0 {
			if err := db.Table("user_models").Where("id = ?", user.ID).Updates(updates).Error; err != nil {
				log.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to update user with encrypted data")
			}
		}
	}

	log.Info().Int("count", len(users)).Msg("Encrypted existing user data")
	return nil
}

func isColumnExistsError(err error) bool {
	// MySQL error code for duplicate column
	return err != nil && (err.Error() == "Error 1060: Duplicate column name" ||
		err.Error() == "Error 1060: Duplicate column name 'encrypted_name'" ||
		err.Error() == "Error 1060: Duplicate column name 'encrypted_email'" ||
		err.Error() == "Error 1060: Duplicate column name 'encrypted_phone_number'")
}
