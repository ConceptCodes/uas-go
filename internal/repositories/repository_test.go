package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"uas/internal/models"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	return gormDB, mock
}

var fixedTime = time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

func matchSQL(s string) string { return regexp.QuoteMeta(s) }

// DepartmentRepository

func TestGormDepartmentRepository_FindById(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormDepartmentRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `department_models` WHERE id = ? AND `department_models`.`deleted_at` IS NULL ORDER BY `department_models`.`id` LIMIT ?")).
		WithArgs("dept-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "name", "secret"}).
			AddRow("dept-1", fixedTime, fixedTime, nil, "Engineering", "secret123"))

	result, err := repo.FindById("dept-1")
	require.NoError(t, err)
	require.Equal(t, "Engineering", result.Name)
}

func TestGormDepartmentRepository_Create(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormDepartmentRepository(gormDB)

	mock.ExpectExec(matchSQL("INSERT INTO `department_models` (`id`,`created_at`,`updated_at`,`deleted_at`,`name`,`secret`) VALUES (?,?,?,?,?,?)")).
		WithArgs("dept-1", sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "Engineering", "secret123").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(&models.DepartmentModel{ID: "dept-1", Name: "Engineering", Secret: "secret123"}))
}

// UserRepository

var userCols = []string{"id", "created_at", "updated_at", "deleted_at", "name", "email", "password", "phone_number", "email_verified", "encrypted_name", "encrypted_email", "encrypted_phone_number"}

func userRow() *sqlmock.Rows {
	return sqlmock.NewRows(userCols).AddRow("user-1", fixedTime, fixedTime, nil, "Alice", "alice@test.com", "hashed", "+1234567890", true, "", "", "")
}

func userJoinCols() string {
	return "SELECT `user_models`.`id`,`user_models`.`created_at`,`user_models`.`updated_at`,`user_models`.`deleted_at`,`user_models`.`name`,`user_models`.`email`,`user_models`.`password`,`user_models`.`phone_number`,`user_models`.`email_verified`,`user_models`.`encrypted_name`,`user_models`.`encrypted_email`,`user_models`.`encrypted_phone_number`"
}

func userJoinFrom() string {
	return " FROM `user_models` JOIN department_roles ON department_roles.user_id = users.id"
}

func TestGormUserRepository_FindById(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormUserRepository(gormDB)

	mock.ExpectQuery(matchSQL(userJoinCols()+userJoinFrom()+" WHERE users.id = ? AND department_roles.id = ? AND `user_models`.`deleted_at` IS NULL ORDER BY `user_models`.`id` LIMIT ?")).
		WithArgs("user-1", "dept-1", 1).
		WillReturnRows(userRow())

	result, err := repo.FindById("user-1", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", result.ID)
}

func TestGormUserRepository_FindByEmail(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormUserRepository(gormDB)

	mock.ExpectQuery(matchSQL(userJoinCols()+userJoinFrom()+" WHERE users.email = ? AND department_roles.id = ? AND `user_models`.`deleted_at` IS NULL ORDER BY `user_models`.`id` LIMIT ?")).
		WithArgs("alice@test.com", "dept-1", 1).
		WillReturnRows(userRow())

	result, err := repo.FindByEmail("alice@test.com", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "alice@test.com", result.Email)
}

func TestGormUserRepository_FindByPhoneNumber(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormUserRepository(gormDB)

	mock.ExpectQuery(matchSQL(userJoinCols()+userJoinFrom()+" WHERE users.phone_number = ? AND department_roles.id = ? AND `user_models`.`deleted_at` IS NULL ORDER BY `user_models`.`id` LIMIT ?")).
		WithArgs("+1234567890", "dept-1", 1).
		WillReturnRows(userRow())

	result, err := repo.FindByPhoneNumber("+1234567890", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "+1234567890", result.PhoneNumber)
}

func TestGormUserRepository_Create(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormUserRepository(gormDB)

	mock.ExpectExec(matchSQL("INSERT INTO `user_models` (`id`,`created_at`,`updated_at`,`deleted_at`,`name`,`email`,`password`,`phone_number`,`email_verified`,`encrypted_name`,`encrypted_email`,`encrypted_phone_number`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)")).
		WithArgs("user-1", sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "Alice", "alice@test.com", "hashed", "+1234567890", true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(&models.UserModel{ID: "user-1", Name: "Alice", Email: "alice@test.com", Password: "hashed", PhoneNumber: "+1234567890", EmailVerified: true}))
}

func TestGormUserRepository_Save(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormUserRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `user_models` SET `created_at`=?,`updated_at`=?,`deleted_at`=?,`name`=?,`email`=?,`password`=?,`phone_number`=?,`email_verified`=?,`encrypted_name`=?,`encrypted_email`=?,`encrypted_phone_number`=? WHERE department_roles.id = ? AND `user_models`.`deleted_at` IS NULL AND `id` = ?")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "Alice", "alice@test.com", "hashed", "+1234567890", true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "dept-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Save(&models.UserModel{
		ID: "user-1", Name: "Alice", Email: "alice@test.com", Password: "hashed",
		PhoneNumber: "+1234567890", EmailVerified: true,
	}, "dept-1"))
}

// SessionRepository

func TestGormSessionRepository_Create(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectExec(matchSQL("INSERT INTO `sessions` (`id`,`created_at`,`updated_at`,`deleted_at`,`user_id`,`department_id`,`refresh_token`,`ip_address`,`user_agent`,`expires_at`,`revoked_at`) VALUES (?,?,?,?,?,?,?,?,?,?,?)")).
		WithArgs("session-1", sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "user-1", "dept-1", sqlmock.AnyArg(), "ip", "agent", sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(&models.Session{
		ID: "session-1", UserID: "user-1", DepartmentID: "dept-1",
		RefreshToken: "plain-token", IPAddress: "ip", UserAgent: "agent",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}))
}

func TestGormSessionRepository_FindByRefreshToken(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `sessions` WHERE (refresh_token = ? AND revoked_at IS NULL AND expires_at > ?) AND department_id = ? AND `sessions`.`deleted_at` IS NULL ORDER BY `sessions`.`id` LIMIT ?")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "dept-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "department_id", "refresh_token", "ip_address", "user_agent", "expires_at", "revoked_at"}).
			AddRow("session-1", fixedTime, fixedTime, nil, "user-1", "dept-1", "hash", "ip", "agent", time.Now().Add(24*time.Hour), nil))

	result, err := repo.FindByRefreshToken("plain-token", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "session-1", result.ID)
}

func TestGormSessionRepository_FindByRefreshToken_LegacyFallback(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `sessions` WHERE (refresh_token = ? AND revoked_at IS NULL AND expires_at > ?) AND department_id = ? AND `sessions`.`deleted_at` IS NULL ORDER BY `sessions`.`id` LIMIT ?")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "dept-1", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	mock.ExpectQuery(matchSQL("SELECT * FROM `sessions` WHERE (refresh_token = ? AND revoked_at IS NULL AND expires_at > ?) AND department_id = ? AND `sessions`.`deleted_at` IS NULL ORDER BY `sessions`.`id` LIMIT ?")).
		WithArgs("plain-token", sqlmock.AnyArg(), "dept-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "department_id", "refresh_token", "ip_address", "user_agent", "expires_at", "revoked_at"}).
			AddRow("session-1", fixedTime, fixedTime, nil, "user-1", "dept-1", "plain-token", "ip", "agent", time.Now().Add(24*time.Hour), nil))

	mock.ExpectExec(matchSQL("UPDATE `sessions` SET `refresh_token`=?,`updated_at`=? WHERE `sessions`.`id` = ? AND `sessions`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "session-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := repo.FindByRefreshToken("plain-token", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "session-1", result.ID)
}

func TestGormSessionRepository_FindByUserID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `sessions` WHERE (user_id = ? AND revoked_at IS NULL AND expires_at > ?) AND department_id = ? AND `sessions`.`deleted_at` IS NULL ORDER BY created_at DESC")).
		WithArgs("user-1", sqlmock.AnyArg(), "dept-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "department_id", "refresh_token", "ip_address", "user_agent", "expires_at", "revoked_at"}).
			AddRow("session-1", fixedTime, fixedTime, nil, "user-1", "dept-1", "token", "ip", "agent", time.Now().Add(24*time.Hour), nil))

	sessions, err := repo.FindByUserID("user-1", "dept-1")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
}

func TestGormSessionRepository_RevokeSession(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `sessions` SET `revoked_at`=?,`updated_at`=? WHERE id = ? AND department_id = ? AND `sessions`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "session-1", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.RevokeSession("session-1", "dept-1"))
}

func TestGormSessionRepository_RevokeAllUserSessions(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `sessions` SET `revoked_at`=?,`updated_at`=? WHERE (user_id = ? AND revoked_at IS NULL) AND department_id = ? AND `sessions`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 3))

	require.NoError(t, repo.RevokeAllUserSessions("user-1", "dept-1"))
}

func TestGormSessionRepository_RevokeAllByDepartment(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `sessions` SET `revoked_at`=?,`updated_at`=? WHERE revoked_at IS NULL AND department_id = ? AND `sessions`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 5))

	require.NoError(t, repo.RevokeAllByDepartment("dept-1"))
}

func TestGormSessionRepository_DeleteExpiredSessions(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSessionRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `sessions` SET `deleted_at`=? WHERE expires_at < ? AND `sessions`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))

	require.NoError(t, repo.DeleteExpiredSessions())
}

// AuthRepository

func TestGormAuthRepository_FindByTokenAndType(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormAuthRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `auth_models` WHERE (token = ? AND type = ?) AND `auth_models`.`deleted_at` IS NULL ORDER BY `auth_models`.`user_id` LIMIT ?")).
		WithArgs(sqlmock.AnyArg(), "reset-password", 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "token", "type", "department_id", "created_at", "updated_at", "deleted_at"}).
			AddRow("user-1", "hashed-token", "reset-password", "dept-1", fixedTime, fixedTime, nil))

	result, err := repo.FindByTokenAndType("plain-token", "reset-password")
	require.NoError(t, err)
	require.Equal(t, "user-1", result.UserID)
}

func TestGormAuthRepository_FindByTokenAndType_LegacyFallback(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormAuthRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `auth_models` WHERE (token = ? AND type = ?) AND `auth_models`.`deleted_at` IS NULL ORDER BY `auth_models`.`user_id` LIMIT ?")).
		WithArgs(sqlmock.AnyArg(), "reset-password", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	mock.ExpectQuery(matchSQL("SELECT * FROM `auth_models` WHERE (token = ? AND type = ?) AND `auth_models`.`deleted_at` IS NULL ORDER BY `auth_models`.`user_id` LIMIT ?")).
		WithArgs("plain-token", "reset-password", 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "token", "type", "department_id", "created_at", "updated_at", "deleted_at"}).
			AddRow("user-1", "plain-token", "reset-password", "dept-1", fixedTime, fixedTime, nil))

	mock.ExpectExec(matchSQL("UPDATE `auth_models` SET `token`=?,`updated_at`=? WHERE `auth_models`.`user_id` = ? AND `auth_models`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := repo.FindByTokenAndType("plain-token", "reset-password")
	require.NoError(t, err)
	require.Equal(t, "user-1", result.UserID)
}

func TestGormAuthRepository_FindByTokenAndTypeScoped(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormAuthRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `auth_models` WHERE (token = ? AND type = ?) AND department_id = ? AND `auth_models`.`deleted_at` IS NULL ORDER BY `auth_models`.`user_id` LIMIT ?")).
		WithArgs(sqlmock.AnyArg(), "reset-password", "dept-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "token", "type", "department_id", "created_at", "updated_at", "deleted_at"}).
			AddRow("user-1", "hashed-token", "reset-password", "dept-1", fixedTime, fixedTime, nil))

	result, err := repo.FindByTokenAndTypeScoped("plain-token", "reset-password", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", result.UserID)
}

func TestGormAuthRepository_Create(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormAuthRepository(gormDB)

	mock.ExpectExec(matchSQL("INSERT INTO `auth_models` (`user_id`,`token`,`type`,`department_id`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?)")).
		WithArgs("user-1", sqlmock.AnyArg(), "reset-password", "dept-1", sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(&models.AuthModel{UserID: "user-1", Token: "plain-token", Type: "reset-password", DepartmentID: "dept-1"}))
}

func TestGormAuthRepository_Delete(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormAuthRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `auth_models` SET `deleted_at`=? WHERE id = ? AND department_id = ? AND `auth_models`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), "auth-1", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Delete("auth-1", "dept-1"))
}

func TestGormAuthRepository_DeleteByTokenAndType(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormAuthRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `auth_models` SET `deleted_at`=? WHERE ((token = ? OR token = ?) AND type = ?) AND department_id = ? AND `auth_models`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "plain-token", "reset-password", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.DeleteByTokenAndType("plain-token", "reset-password", "dept-1"))
}

// DepartmentRoleRepository

func TestGormDepartmentRoleRepository_FindById(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormDepartmentRoleRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `department_roles` WHERE (id = ? AND userId = ?) AND `department_roles`.`deleted_at` IS NULL ORDER BY `department_roles`.`id` LIMIT ?")).
		WithArgs("dept-1", "user-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "user_id", "created_at", "updated_at", "deleted_at"}).
			AddRow("dept-1", "admin", "user-1", fixedTime, fixedTime, nil))

	result, err := repo.FindById("dept-1", "user-1")
	require.NoError(t, err)
	require.Equal(t, "dept-1", result.ID)
}

func TestGormDepartmentRoleRepository_FindByUserID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormDepartmentRoleRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `department_roles` WHERE (user_id = ? AND id = ?) AND `department_roles`.`deleted_at` IS NULL ORDER BY updated_at DESC,`department_roles`.`id` LIMIT ?")).
		WithArgs("user-1", "dept-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "user_id", "created_at", "updated_at", "deleted_at"}).
			AddRow("dept-1", "admin", "user-1", fixedTime, fixedTime, nil))

	result, err := repo.FindByUserID("user-1", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "admin", string(result.Role))
}

func TestGormDepartmentRoleRepository_Create(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormDepartmentRoleRepository(gormDB)

	mock.ExpectExec(matchSQL("INSERT INTO `department_roles` (`id`,`role`,`user_id`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?)")).
		WithArgs("dept-1", "admin", "user-1", sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(&models.DepartmentRoles{ID: "dept-1", Role: "admin", UserID: "user-1"}))
}

func TestGormDepartmentRoleRepository_Update(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormDepartmentRoleRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `department_roles` SET `role`=?,`created_at`=?,`updated_at`=?,`deleted_at`=? WHERE `department_roles`.`deleted_at` IS NULL AND `id` = ? AND `user_id` = ?")).
		WithArgs("admin", sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "dept-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Update(&models.DepartmentRoles{ID: "dept-1", Role: "admin", UserID: "user-1"}))
}

// MfaFactorRepository

func TestGormMfaFactorRepository_FindByID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormMfaFactorRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `mfa_factors` WHERE id = ? AND department_id = ? ORDER BY `mfa_factors`.`id` LIMIT ?")).
		WithArgs("factor-1", "dept-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "department_id", "factor_type", "secret", "name", "is_primary", "backup_codes", "last_used_at", "created_at", "updated_at"}).
			AddRow("factor-1", "user-1", "dept-1", "totp", "secret", "My App", false, "", nil, fixedTime, fixedTime))

	result, err := repo.FindByID("factor-1", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "totp", string(result.FactorType))
}

func TestGormMfaFactorRepository_CountActiveByUserID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormMfaFactorRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT count(*) FROM `mfa_factors` WHERE (user_id = ? AND deleted_at IS NULL) AND department_id = ?")).
		WithArgs("user-1", "dept-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := repo.CountActiveByUserID("user-1", "dept-1")
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
}

func TestGormMfaFactorRepository_SetPrimary(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormMfaFactorRepository(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec(matchSQL("UPDATE `mfa_factors` SET `is_primary`=?,`updated_at`=? WHERE user_id = ? AND department_id = ?")).
		WithArgs(false, sqlmock.AnyArg(), "user-1", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(matchSQL("UPDATE `mfa_factors` SET `is_primary`=?,`updated_at`=? WHERE id = ? AND department_id = ?")).
		WithArgs(true, sqlmock.AnyArg(), "factor-1", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SetPrimary("factor-1", "user-1", "dept-1"))
}

func TestGormMfaFactorRepository_UpdateBackupCodes(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormMfaFactorRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `mfa_factors` SET `backup_codes`=?,`updated_at`=? WHERE id = ? AND department_id = ?")).
		WithArgs("code1,code2", sqlmock.AnyArg(), "factor-1", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateBackupCodes("factor-1", "dept-1", "code1,code2"))
}

func TestGormMfaFactorRepository_UpdateLastUsed(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormMfaFactorRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `mfa_factors` SET `last_used_at`=?,`updated_at`=? WHERE id = ? AND department_id = ?")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "factor-1", "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateLastUsed("factor-1", "dept-1"))
}

// MfaChallengeRepository

func TestGormMfaChallengeRepository_Create(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormMfaChallengeRepository(gormDB)

	mock.ExpectExec(matchSQL("INSERT INTO `mfa_challenges` (`id`,`user_id`,`department_id`,`factor_id`,`challenge_type`,`state`,`temp_token`,`code`,`expires_at`,`created_at`) VALUES (?,?,?,?,?,?,?,?,?,?)")).
		WithArgs("chal-1", "user-1", "dept-1", nil, "totp", "pending", "temp-token", "123456", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(&models.MfaChallenge{
		ID: "chal-1", UserID: "user-1", DepartmentID: "dept-1",
		ChallengeType: "totp", State: "pending", TempToken: "temp-token", Code: "123456",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}))
}

func TestGormMfaChallengeRepository_VerifyChallenge(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormMfaChallengeRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `mfa_challenges` SET `state`=? WHERE id = ?")).
		WithArgs("verified", "chal-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.VerifyChallenge("chal-1"))
}

// PasswordHistoryRepository

func TestGormPasswordHistoryRepository_GetRecentPasswords(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormPasswordHistoryRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `password_histories` WHERE user_id = ? AND department_id = ? AND `password_histories`.`deleted_at` IS NULL ORDER BY created_at DESC LIMIT ?")).
		WithArgs("user-1", "dept-1", 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "department_id", "password_hash"}).
			AddRow("ph-1", fixedTime, fixedTime, nil, "user-1", "dept-1", "hash1").
			AddRow("ph-2", fixedTime, fixedTime, nil, "user-1", "dept-1", "hash2"))

	results, err := repo.GetRecentPasswords("user-1", "dept-1", 5)
	require.NoError(t, err)
	require.Len(t, results, 2)
}

func TestGormPasswordHistoryRepository_DeleteOldPasswords_UnderLimit(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormPasswordHistoryRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT count(*) FROM `password_histories` WHERE user_id = ? AND department_id = ? AND `password_histories`.`deleted_at` IS NULL")).
		WithArgs("user-1", "dept-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	require.NoError(t, repo.DeleteOldPasswords("user-1", "dept-1", 5))
}

func TestGormPasswordHistoryRepository_DeleteOldPasswords_OverLimit(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormPasswordHistoryRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT count(*) FROM `password_histories` WHERE user_id = ? AND department_id = ? AND `password_histories`.`deleted_at` IS NULL")).
		WithArgs("user-1", "dept-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	mock.ExpectExec(matchSQL("UPDATE `password_histories` SET `deleted_at`=? WHERE user_id = ? AND department_id = ? AND `password_histories`.`deleted_at` IS NULL ORDER BY created_at DESC OFFSET ?")).
		WithArgs(sqlmock.AnyArg(), "user-1", "dept-1", 5).
		WillReturnResult(sqlmock.NewResult(0, 5))

	require.NoError(t, repo.DeleteOldPasswords("user-1", "dept-1", 5))
}

func TestGormPasswordHistoryRepository_DeleteByDepartment(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormPasswordHistoryRepository(gormDB)

	mock.ExpectExec(matchSQL("UPDATE `password_histories` SET `deleted_at`=? WHERE department_id = ? AND `password_histories`.`deleted_at` IS NULL")).
		WithArgs(sqlmock.AnyArg(), "dept-1").
		WillReturnResult(sqlmock.NewResult(0, 10))

	require.NoError(t, repo.DeleteByDepartment("dept-1"))
}

// IdentityProviderRepository

func TestGormIdentityProviderRepository_FindByID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormIdentityProviderRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `identity_providers` WHERE id = ? AND department_id = ? ORDER BY `identity_providers`.`id` LIMIT ?")).
		WithArgs("idp-1", "dept-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "department_id", "name", "provider_type", "client_id", "client_secret", "issuer_url", "authorization_url", "token_url", "user_info_url", "jwks_uri", "metadata_url", "redirect_urls", "scopes", "enabled", "created_at", "updated_at"}).
			AddRow("idp-1", "dept-1", "Google", "oidc", "client-id", "", "https://issuer", "", "", "", "", "", "[]", "openid", true, fixedTime, fixedTime))

	result, err := repo.FindByID("idp-1", "dept-1")
	require.NoError(t, err)
	require.Equal(t, "Google", result.Name)
}

// SecurityAuditRepository

func TestGormSecurityAuditRepository_Create(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSecurityAuditRepository(gormDB)

	mock.ExpectExec(matchSQL("INSERT INTO `security_events` (`id`,`created_at`,`updated_at`,`deleted_at`,`event_type`,`user_id`,`department_id`,`ip_address`,`user_agent`,`status`,`error_message`,`request_id`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "login_failed", "user-1", "dept-1", "127.0.0.1", "agent", "failure", "bad password", "req-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(&models.SecurityEvent{
		EventType: "login_failed", UserID: "user-1", DepartmentID: "dept-1",
		IPAddress: "127.0.0.1", UserAgent: "agent", Status: "failure",
		ErrorMessage: "bad password", RequestID: "req-1",
	}))
}

func TestGormSecurityAuditRepository_GetByUserID(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormSecurityAuditRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `security_events` WHERE user_id = ? AND `security_events`.`deleted_at` IS NULL ORDER BY created_at DESC LIMIT ?")).
		WithArgs("user-1", 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "event_type", "user_id", "department_id", "ip_address", "user_agent", "status", "error_message", "request_id"}).
			AddRow("evt-1", fixedTime, fixedTime, nil, "login_failed", "user-1", "dept-1", "127.0.0.1", "agent", "failure", "bad password", "req-1"))

	results, err := repo.GetByUserID("user-1", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

// WebhookEndpointRepository

func TestGormWebhookEndpointRepository_FindActiveByDepartmentAndEvent(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormWebhookEndpointRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `webhook_endpoints` WHERE (is_active = ? AND JSON_CONTAINS(events, ?)) AND department_id = ?")).
		WithArgs(true, `"user.created"`, "dept-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "department_id", "name", "url", "secret", "events", "is_active", "created_at", "updated_at"}).
			AddRow("wh-1", "dept-1", "My Webhook", "https://hook", "secret123", `["user.created"]`, true, fixedTime, fixedTime))

	results, err := repo.FindActiveByDepartmentAndEvent("dept-1", "user.created")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "My Webhook", results[0].Name)
}

// WebhookDeliveryRepository

func TestGormWebhookDeliveryRepository_FindPendingRetries(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormWebhookDeliveryRepository(gormDB)

	mock.ExpectQuery(matchSQL("SELECT * FROM `webhook_deliveries` WHERE status = ? AND next_retry_at <= ? AND attempt < max_attempts")).
		WithArgs("pending", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "endpoint_id", "department_id", "event", "payload", "response_code", "response_body", "status", "attempt", "max_attempts", "next_retry_at", "created_at", "updated_at"}).
			AddRow("del-1", "wh-1", "dept-1", "user.created", "{}", 0, "", "pending", 2, 5, fixedTime, fixedTime, fixedTime))

	results, err := repo.FindPendingRetries()
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGormWebhookDeliveryRepository_DeleteOlderThan(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewGormWebhookDeliveryRepository(gormDB)

	mock.ExpectExec(matchSQL("DELETE FROM `webhook_deliveries` WHERE created_at < ?")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 100))

	count, err := repo.DeleteOlderThan(30 * 24 * time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(100), count)
}
