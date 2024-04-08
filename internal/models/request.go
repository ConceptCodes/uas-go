package models

type Request struct {
	Id       string
	TenantId string
	User     UserModel
}

type OnboardTenantRequest struct {
	Name         string `json:"name"`
	DepartmentID string `json:"department_id"`
}

type CredentialsRegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
