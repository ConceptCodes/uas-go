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

type CredentialsRegisterRequest struct {
	Name     string `json:"name" validate:"noSQLKeywords"`
	Email    string `json:"email" validate:"email,required,noSQLKeywords"`
	Password string `json:"password" validate:"required,noSQLKeywords"`
}
