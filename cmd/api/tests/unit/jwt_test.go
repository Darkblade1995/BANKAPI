package unit_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret-key-123456"

type Claims struct {
	UserId int `json:"userId"`
	jwt.RegisteredClaims
}

func generateTestToken(userId int, secret string, duration time.Duration) (string, error) {
	claims := Claims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func validateTestToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

func TestGenerateToken_Success(t *testing.T) {
	token, err := generateTestToken(1, testSecret, 15*time.Minute)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_Success(t *testing.T) {
	token, _ := generateTestToken(1, testSecret, 15*time.Minute)

	claims, err := validateTestToken(token, testSecret)

	assert.NoError(t, err)
	assert.Equal(t, 1, claims.UserId)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	token, _ := generateTestToken(1, testSecret, -1*time.Minute)

	_, err := validateTestToken(token, testSecret)

	assert.Error(t, err)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, _ := generateTestToken(1, testSecret, 15*time.Minute)

	_, err := validateTestToken(token, "wrong-secret")

	assert.Error(t, err)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := validateTestToken("token.invalido.completamente", testSecret)

	assert.Error(t, err)
}