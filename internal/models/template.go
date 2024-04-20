package models

type ForgotPasswordData struct {
	Name string
	Url  string
}

type VerifyEmailData struct {
	Name string
	Otp  string
}
