package models

type Request struct {
	ID           string
	DepartmentID string
	User         UserModel
}

type OnboardTenantRequest struct {
	DepartmentName string `json:"departmentName" validate:"required,noSQLKeywords"`
	DepartmentID   string `json:"departmentId" validate:"required,noSQLKeywords"`
}

type CredentialsLoginRequest struct {
	Email    string `json:"email" validate:"email,required,noSQLKeywords"`
	Password string `json:"password" validate:"required,noSQLKeywords"`
}

type RegisterRequest struct {
	Name        string `json:"name" validate:"required,noSQLKeywords"`
	Email       string `json:"email" validate:"email,required,noSQLKeywords"`
	Password    string `json:"password" validate:"required,noSQLKeywords"`
	PhoneNumber string `json:"phoneNumber" validate:"required,e164,noSQLKeywords"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"email,required,noSQLKeywords"`
}

type ResetPasswordRequest struct {
	Password string `json:"password" validate:"email,required,noSQLKeywords"`
}

type SendOtpRequest struct {
	PhoneNumber string `json:"phoneNumber" validate:"required,noSQLKeywords"`
}

type VerifyOtpRequest struct {
	PhoneNumber string `json:"phoneNumber" validate:"required,noSQLKeywords"`
	Otp         string `json:"otp" validate:"required,noSQLKeywords,numeric"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"email,required,noSQLKeywords"`
	Otp   string `json:"otp" validate:"required,noSQLKeywords,numeric"`
}

type MagicLinkEmailRequest = ForgotPasswordRequest