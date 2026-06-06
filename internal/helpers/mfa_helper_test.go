package helpers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMfaAmrClaimForFactor(t *testing.T) {
	h := &MfaHelper{}

	assert.Equal(t, "otp", h.AmrClaimForFactor("totp"))
	assert.Equal(t, "sms", h.AmrClaimForFactor("sms"))
	assert.Equal(t, "email", h.AmrClaimForFactor("email"))
	assert.Equal(t, "mfa", h.AmrClaimForFactor("unknown"))
	assert.Equal(t, "mfa", h.AmrClaimForFactor(""))
}

func TestMfaHashBackupCode(t *testing.T) {
	h := &MfaHelper{}

	hash := h.HashBackupCode("12345-67890")
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)

	hash2 := h.HashBackupCode("12345-67890")
	assert.Equal(t, hash, hash2)

	diff := h.HashBackupCode("54321-09876")
	assert.NotEqual(t, hash, diff)
}

func TestMfaVerifyBackupCode(t *testing.T) {
	h := &MfaHelper{}

	hashedCodes := []string{
		h.HashBackupCode("11111-11111"),
		h.HashBackupCode("22222-22222"),
		h.HashBackupCode("33333-33333"),
	}
	hashedJSON, _ := json.Marshal(hashedCodes)

	valid, remainingJSON, err := h.VerifyBackupCode("22222-22222", string(hashedJSON))
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, remainingJSON)

	var remaining []string
	err = json.Unmarshal([]byte(remainingJSON), &remaining)
	require.NoError(t, err)
	assert.Len(t, remaining, 2)
	assert.NotContains(t, remaining, h.HashBackupCode("22222-22222"))
}

func TestMfaVerifyBackupCodeInvalid(t *testing.T) {
	h := &MfaHelper{}

	hashedCodes := []string{h.HashBackupCode("11111-11111")}
	hashedJSON, _ := json.Marshal(hashedCodes)

	valid, remaining, err := h.VerifyBackupCode("99999-99999", string(hashedJSON))
	require.NoError(t, err)
	assert.False(t, valid)
	assert.Empty(t, remaining)
}

func TestMfaVerifyBackupCodeInvalidJSON(t *testing.T) {
	h := &MfaHelper{}

	_, _, err := h.VerifyBackupCode("11111-11111", "not-json")
	assert.Error(t, err)
}

func TestMfaGenerateBackupCodes(t *testing.T) {
	h := &MfaHelper{}

	codes, err := h.GenerateBackupCodes(10)
	require.NoError(t, err)
	assert.Len(t, codes, 10)

	for _, code := range codes {
		assert.Len(t, code, 11)
		assert.Contains(t, code, "-")
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "duplicate backup code: %s", code)
		seen[code] = true
	}
}

func TestMfaGenerateBackupCodesZeroCount(t *testing.T) {
	h := &MfaHelper{}

	codes, err := h.GenerateBackupCodes(0)
	require.NoError(t, err)
	assert.Empty(t, codes)
}

func TestMfaGenerateNonce(t *testing.T) {
	h := &MfaHelper{}

	nonce1, err := h.GenerateNonce()
	require.NoError(t, err)
	assert.NotEmpty(t, nonce1)

	nonce2, err := h.GenerateNonce()
	require.NoError(t, err)
	assert.NotEqual(t, nonce1, nonce2)
}

func TestMfaGenerateCodeVerifier(t *testing.T) {
	h := &MfaHelper{}

	verifier, challenge, err := h.GenerateCodeVerifier()
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.NotEmpty(t, challenge)
	assert.NotEqual(t, verifier, challenge)
}

func TestMfaGetTOTPQRCodeURI(t *testing.T) {
	h := &MfaHelper{}

	uri := h.GetTOTPQRCodeURI("JBSWY3DPEHPK3PXP", "test@example.com")
	assert.NotEmpty(t, uri)
	assert.Contains(t, uri, "otpauth://")
	assert.Contains(t, uri, "test@example.com")
}

func TestMfaValidateTOTPCodeInvalid(t *testing.T) {
	h := &MfaHelper{}

	valid := h.ValidateTOTPCode("JBSWY3DPEHPK3PXP", "000000")
	assert.False(t, valid)
}

func TestCheckFactorBelongsToUser(t *testing.T) {
	h := &MfaHelper{}

	result := h.CheckFactorBelongsToUser("factor-1", "user-1", "dept-1", nil)
	assert.True(t, result)
}
