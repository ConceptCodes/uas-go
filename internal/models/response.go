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
