package config

import "testing"

func TestValidateConfig_EmailProvider(t *testing.T) {
	original := AppConfig
	t.Cleanup(func() {
		AppConfig = original
	})

	base := original
	base.RefreshJwtSecret = "12345678901234567890123456789012"
	base.AccessJwtSecret = "12345678901234567890123456789012"
	base.CookieBlockKey = "12345678901234567890123456789012"
	base.CookieHashKey = "12345678901234567890123456789012"
	base.EmailFrom = "noreply@example.com"
	base.Env = "development"

	tests := []struct {
		name          string
		emailProvider string
		expectErr     bool
	}{
		{name: "resend provider", emailProvider: "resend", expectErr: false},
		{name: "mock provider", emailProvider: "mock", expectErr: false},
		{name: "invalid provider", emailProvider: "smtp", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AppConfig = base
			AppConfig.EmailProvider = tt.emailProvider

			err := validateConfig()
			if tt.expectErr && err == nil {
				t.Fatalf("expected error for provider %q, got nil", tt.emailProvider)
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error for provider %q, got %v", tt.emailProvider, err)
			}
		})
	}
}
