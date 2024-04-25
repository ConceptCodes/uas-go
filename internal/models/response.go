package models

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResponse struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

type OnboardDepartmentResponse struct {
	DepartmentID   string `json:"departmentId"`
	DepartmentName string `json:"departmentName"`
}

type HealthCheckResponse struct {
	Service string `json:"service"`
	Status  bool   `json:"status"`
}

type RegisterUserResponse struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}
