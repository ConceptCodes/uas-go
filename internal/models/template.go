package models

type ForgotPasswordData struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}
