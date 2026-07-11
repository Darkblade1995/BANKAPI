package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func isValidCurrency(currency string) bool {
	valid := map[string]bool{
		"COP": true,
		"USD": true,
		"EUR": true,
	}
	return valid[currency]
}

func isValidAmount(amount int64) bool {
	return amount > 0
}

func isValidEmail(email string) bool {
	if len(email) < 5 {
		return false
	}
	hasAt := false
	hasDot := false
	for _, c := range email {
		if c == '@' {
			hasAt = true
		}
		if c == '.' {
			hasDot = true
		}
	}
	return hasAt && hasDot
}

func isValidPassword(password string) bool {
	return len(password) >= 8
}

// ─── Currency ──────

func TestIsValidCurrency_COP(t *testing.T) {
	assert.True(t, isValidCurrency("COP"))
}

func TestIsValidCurrency_USD(t *testing.T) {
	assert.True(t, isValidCurrency("USD"))
}

func TestIsValidCurrency_EUR(t *testing.T) {
	assert.True(t, isValidCurrency("EUR"))
}

func TestIsValidCurrency_Invalid(t *testing.T) {
	assert.False(t, isValidCurrency("XYZ"))
}

func TestIsValidCurrency_Empty(t *testing.T) {
	assert.False(t, isValidCurrency(""))
}

// ─── Amount ────────

func TestIsValidAmount_Positive(t *testing.T) {
	assert.True(t, isValidAmount(100000))
}

func TestIsValidAmount_Zero(t *testing.T) {
	assert.False(t, isValidAmount(0))
}

func TestIsValidAmount_Negative(t *testing.T) {
	assert.False(t, isValidAmount(-100))
}

// ─── Email ────

func TestIsValidEmail_Valid(t *testing.T) {
	assert.True(t, isValidEmail("fernando@gmail.com"))
}

func TestIsValidEmail_NoAt(t *testing.T) {
	assert.False(t, isValidEmail("fernandogmail.com"))
}

func TestIsValidEmail_TooShort(t *testing.T) {
	assert.False(t, isValidEmail("a@b"))
}

// ─── Password ─────

func TestIsValidPassword_Valid(t *testing.T) {
	assert.True(t, isValidPassword("12345678"))
}

func TestIsValidPassword_TooShort(t *testing.T) {
	assert.False(t, isValidPassword("123"))
}

func TestIsValidPassword_Exactly8(t *testing.T) {
	assert.True(t, isValidPassword("12345678"))
}