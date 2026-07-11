package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func convertCurrency(amount int64, fromCurrency, toCurrency string, rates map[string]float64) (int64, float64, error) {
	if fromCurrency == toCurrency {
		return amount, 1.0, nil
	}

	rate, exists := rates[toCurrency]
	if !exists {
		return 0, 0, assert.AnError
	}

	converted := int64(float64(amount) * rate)
	return converted, rate, nil
}

func TestConvert_SameCurrency(t *testing.T) {
	amount, rate, err := convertCurrency(100000, "COP", "COP", nil)

	assert.NoError(t, err)
	assert.Equal(t, int64(100000), amount)
	assert.Equal(t, 1.0, rate)
}

func TestConvert_COPtoUSD(t *testing.T) {
	rates := map[string]float64{
		"USD": 0.00028,
		"EUR": 0.00024,
	}

	converted, rate, err := convertCurrency(1000000, "COP", "USD", rates)

	assert.NoError(t, err)
	assert.Equal(t, int64(280), converted)
	assert.Equal(t, 0.00028, rate)
}

func TestConvert_COPtoEUR(t *testing.T) {
	rates := map[string]float64{
		"USD": 0.00028,
		"EUR": 0.00024,
	}

	converted, rate, err := convertCurrency(1000000, "COP", "EUR", rates)

	assert.NoError(t, err)
	assert.Equal(t, int64(240), converted)
	assert.Equal(t, 0.00024, rate)
}

func TestConvert_UnsupportedCurrency(t *testing.T) {
	rates := map[string]float64{
		"USD": 0.00028,
	}

	_, _, err := convertCurrency(100000, "COP", "XYZ", rates)

	assert.Error(t, err)
}

func TestConvert_ZeroAmount(t *testing.T) {
	rates := map[string]float64{
		"USD": 0.00028,
	}

	converted, _, err := convertCurrency(0, "COP", "USD", rates)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), converted)
}