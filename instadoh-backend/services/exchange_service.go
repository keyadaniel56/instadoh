package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

// ExchangeRate holds rate information for a single currency
type ExchangeRate struct {
	Currency string  `json:"currency"`
	Rate     float64 `json:"rate"`
}

// ExchangeRateResponse from exchangerate-api.com
type ExchangeRateResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// ExchangeService provides real-time and cached exchange rates
type ExchangeService struct {
	baseURL    string
	apiKey     string
	cacheTTL   int
	cache      map[string]*cachedRate
	mu         sync.RWMutex
	httpClient *http.Client
}

type cachedRate struct {
	rate      float64
	expiresAt time.Time
}

// BTCBaseRate is the USD price of 1 BTC (will be fetched dynamically)
const BTCBaseRate = 0 // Placeholder - would fetch from CoinGecko etc.

// NewExchangeService initializes the exchange rate service
func NewExchangeService(baseURL, apiKey string, cacheTTL int) *ExchangeService {
	svc := &ExchangeService{
		baseURL:  baseURL,
		apiKey:   apiKey,
		cacheTTL: cacheTTL,
		cache:    make(map[string]*cachedRate),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Start background cache refresh
	go svc.backgroundRefresh()

	return svc
}

// GetRate returns the exchange rate from BTC to the target currency
// Returns how much 1 BTC is worth in the target currency
func (s *ExchangeService) GetRate(currency string) (float64, error) {
	if currency == "BTC" {
		return 1.0, nil
	}

	// Check cache first
	s.mu.RLock()
	cached, exists := s.cache[currency]
	s.mu.RUnlock()

	if exists && time.Now().Before(cached.expiresAt) {
		return cached.rate, nil
	}

	// Fetch fresh rate
	rate, err := s.fetchRate(currency)
	if err != nil {
		// Return stale cache if fetch fails
		if exists {
			log.Printf("Warning: Failed to fetch fresh rate for %s, using cached: %v", currency, err)
			return cached.rate, nil
		}
		return 0, fmt.Errorf("failed to fetch rate for %s: %w", currency, err)
	}

	// Update cache
	s.mu.Lock()
	s.cache[currency] = &cachedRate{
		rate:      rate,
		expiresAt: time.Now().Add(time.Duration(s.cacheTTL) * time.Second),
	}
	s.mu.Unlock()

	return rate, nil
}

// GetBTCUSDPrice fetches the current BTC/USD price
func (s *ExchangeService) GetBTCUSDPrice() (float64, error) {
	// Using CoinGecko free API
	resp, err := s.httpClient.Get("https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd")
	if err != nil {
		return 0, fmt.Errorf("failed to fetch BTC price: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode BTC price: %w", err)
	}

	price, ok := result["bitcoin"]["usd"]
	if !ok {
		return 0, fmt.Errorf("BTC price not found in response")
	}

	return price, nil
}

// Convert converts an amount from one currency to another
// GetRate returns "how many units of target currency per 1 USD" (e.g., KES 150 = $1)
func (s *ExchangeService) Convert(amount float64, fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}

	// Step 1: Convert fromCurrency to USD
	// If 1 USD = 150 KES (rate), then 100 KES = 100 / 150 = 0.667 USD
	var usdAmount float64
	if fromCurrency == "USD" {
		usdAmount = amount
	} else {
		rateToUSD, err := s.GetRate(fromCurrency)
		if err != nil {
			return 0, fmt.Errorf("failed to get rate for %s: %w", fromCurrency, err)
		}
		// rateToUSD is "how many units of fromCurrency per 1 USD"
		// So amount / rateToUSD gives the USD equivalent
		usdAmount = amount / rateToUSD
	}

	// Step 2: Convert USD to target currency
	// If 1 USD = 150 KES (rate), then 0.667 USD = 0.667 * 150 = 100 KES
	if toCurrency == "USD" {
		return usdAmount, nil
	}

	rateFromUSD, err := s.GetRate(toCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get rate for %s: %w", toCurrency, err)
	}

	return usdAmount * rateFromUSD, nil
}

// GetSupportedCurrencies returns all currencies currently in cache
func (s *ExchangeService) GetSupportedCurrencies() []string {
	supportedCurrencies := []string{"USD", "EUR", "GBP", "JPY", "CNY"}

	// Add currencies from cache
	s.mu.RLock()
	for currency := range s.cache {
		supportedCurrencies = append(supportedCurrencies, currency)
	}
	s.mu.RUnlock()

	return supportedCurrencies
}

func (s *ExchangeService) fetchRate(currency string) (float64, error) {
	// The exchangerate-api.com/v4/latest/{baseCurrency} returns rates with that baseCurrency as base.
	// If we query /latest/USD, the base is USD and rates contain all currencies.
	// If we query /latest/KES, the base is KES and rates contain conversions like "USD": 0.0077.
	// Since the API returns rates relative to the base currency, we need to query with USD as base
	// to get "how many units of target currency per 1 USD".
	url := fmt.Sprintf("%s/USD", s.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("exchange API returned status %d", resp.StatusCode)
	}

	var result ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	// The response has base="USD" and rates contains entries like "KES": 150.0 (KES per 1 USD)
	rate, ok := result.Rates[currency]
	if !ok {
		if result.Base == currency {
			return 1.0, nil
		}
		return 0, fmt.Errorf("rate for %s not found", currency)
	}

	return rate, nil
}

func (s *ExchangeService) backgroundRefresh() {
	ticker := time.NewTicker(time.Duration(s.cacheTTL) * time.Second)
	defer ticker.Stop()

	// Refresh major currencies every interval by fetching them all in one API call
	// The API returns ALL rates when queried with a single base currency
	for range ticker.C {
		// Fetch all rates relative to USD in one API call
		url := fmt.Sprintf("%s/USD", s.baseURL)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Failed to create rate refresh request: %v", err)
			continue
		}

		if s.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.apiKey)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			log.Printf("Failed to refresh rates: %v", err)
			continue
		}

		var result ExchangeRateResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			log.Printf("Failed to decode rate refresh response: %v", err)
			continue
		}
		resp.Body.Close()

		now := time.Now()
		expiry := now.Add(time.Duration(s.cacheTTL) * time.Second)

		s.mu.Lock()
		for currency, rate := range result.Rates {
			s.cache[currency] = &cachedRate{
				rate:      rate,
				expiresAt: expiry,
			}
		}
		// Also cache USD as 1.0
		s.cache["USD"] = &cachedRate{
			rate:      1.0,
			expiresAt: expiry,
		}
		s.mu.Unlock()

		log.Printf("Exchange rates refreshed for %d currencies", len(result.Rates))
	}
}

// FormatCurrency formats a fiat amount according to currency conventions
func FormatCurrency(amount float64, currency string) string {
	precision := 2
	switch currency {
	case "JPY", "KRW", "CLP", "IDR":
		precision = 0
	case "BHD", "KWD", "OMR", "JOD":
		precision = 3
	}

	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, math.Round(amount*math.Pow10(precision))/math.Pow10(precision))
}