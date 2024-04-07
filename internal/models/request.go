package models

type Request struct {
	Id string
}

type OnboardTenantRequest struct {
	Name         string `json:"name"`
	DepartmentID string `json:"department_id"`
}
