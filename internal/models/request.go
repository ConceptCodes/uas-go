package models

type Request struct {
	ID           string
	DepartmentID string
	User         UserModel
}

type OnboardTenantRequest struct {
	DepartmentName string `json:"department_name" validate:"required,noSQLKeywords"`
	DepartmentID   string `json:"department_id" validate:"required,noSQLKeywords"`
}

type CredentialsLoginRequest struct {
	Email    string `json:"email" validate:"email,required,noSQLKeywords"`
	Password string `json:"password" validate:"required,noSQLKeywords"`
}

type RegisterRequest struct {
	Name        string `json:"name" validate:"required,noSQLKeywords"`
	Email       string `json:"email" validate:"email,required,noSQLKeywords"`
	Password    string `json:"password" validate:"required,noSQLKeywords"`
	PhoneNumber string `json:"phone_number" validate:"required,noSQLKeywords"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"email,required,noSQLKeywords"`
}

type ResetPasswordRequest struct {
	Password string `json:"password" validate:"email,required,noSQLKeywords"`
}

type SendOtpRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,noSQLKeywords"`
}

type VerifyOtpRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,noSQLKeywords"`
	Otp         string `json:"otp" validate:"required,noSQLKeywords"`
}
