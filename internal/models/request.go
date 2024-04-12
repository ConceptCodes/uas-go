package models

type Request struct {
	Id       string
	TenantId string
	User     UserModel
}

type OnboardTenantRequest struct {
	Name         string `json:"name" validate:"required,noSQLKeywords"`
	DepartmentID string `json:"department_id" validate:"required,noSQLKeywords"`
}

type CredentialsRequest struct {
	Name     string `json:"name" validate:"noSQLKeywords"`
	Email    string `json:"email" validate:"email,required,noSQLKeywords"`
	Password string `json:"password" validate:"required,noSQLKeywords"`
}
