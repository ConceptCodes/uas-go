package models

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResponse struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

type OnboardTenantResponse struct {
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	TenantSecret   string `json:"tenant_secret"`
}

type HealthCheckResponse struct {
	Service string `json:"service"`
	Status  bool   `json:"status"`
}

type RegisterUserResponse struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}
