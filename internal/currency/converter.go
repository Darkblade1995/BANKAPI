package currency

import (
	"context"
	"fmt"
	"net/http"
	"encoding/json"

	"BANKAPI/internal/cache"
)

type ExchangeRates struct {
	BaseCode        string             `json:"base_code"`
	ConversionRates map[string]float64 `json:"conversion_rates"`
}

type Converter struct {
	apiKey string
	cache  *cache.Cache
}

func NewConverter(apiKey string, cache *cache.Cache) *Converter {
	return &Converter{
		apiKey: apiKey,
		cache:  cache,
	}
}

func (c *Converter) GetRates(baseCurrency string) (map[string]float64, error) {
	ctx := context.Background()

	
	rates, err := c.cache.GetExchangeRates(ctx, baseCurrency)
	if err == nil {
		return rates, nil
	}

	
	url := fmt.Sprintf(
		"https://v6.exchangerate-api.com/v6/%s/latest/%s",
		c.apiKey,
		baseCurrency,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var exchangeRates ExchangeRates
	if err := json.NewDecoder(resp.Body).Decode(&exchangeRates); err != nil {
		return nil, err
	}

	
	c.cache.SetExchangeRates(ctx, baseCurrency, exchangeRates.ConversionRates)

	return exchangeRates.ConversionRates, nil
}

func (c *Converter) Convert(amount int64, fromCurrency, toCurrency string) (int64, float64, error) {
	if fromCurrency == toCurrency {
		return amount, 1.0, nil
	}

	rates, err := c.GetRates(fromCurrency)
	if err != nil {
		return 0, 0, err
	}

	rate, exists := rates[toCurrency]
	if !exists {
		return 0, 0, fmt.Errorf("currency %s not supported", toCurrency)
	}

	convertedAmount := int64(float64(amount) * rate)

	return convertedAmount, rate, nil
}