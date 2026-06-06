package models

type ForgotPasswordData struct {
	Name  string
	Url   string
	Token string
}

type VerifyEmailData struct {
	Name string
	Otp  string
}

type MagicEmailData = ForgotPasswordData
