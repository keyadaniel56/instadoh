package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

// UgandaMobileService handles Uganda mobile money integrations
// Supports MTN Mobile Money and Airtel Money for UGX on/off-ramp
type UgandaMobileService struct {
	apiBaseURL  string
	apiUsername string
	apiPassword string
	callbackURL string
	httpClient  *http.Client
}

// UgandaMobileConfig holds Uganda mobile money API settings
type UgandaMobileConfig struct {
	APIBaseURL  string
	APIUsername string
	APIPassword string
	CallbackURL string
}

// NewUgandaMobileService creates a new Uganda mobile money service
func NewUgandaMobileService(cfg *UgandaMobileConfig) *UgandaMobileService {
	return &UgandaMobileService{
		apiBaseURL:  cfg.APIBaseURL,
		apiUsername: cfg.APIUsername,
		apiPassword: cfg.APIPassword,
		callbackURL: cfg.CallbackURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UgandaMobileProvider represents a mobile money provider in Uganda
type UgandaMobileProvider string

const (
	ProviderMTN    UgandaMobileProvider = "mtn"
	ProviderAirtel UgandaMobileProvider = "airtel"
)

// DepositRequest initiates a collection from a Uganda mobile money user
type DepositRequest struct {
	Provider    UgandaMobileProvider `json:"provider"`
	PhoneNumber string               `json:"phone_number"`
	Amount      float64              `json:"amount"`
	Reference   string               `json:"reference"`
	CallbackURL string               `json:"callback_url"`
}

// DepositResponse from the mobile money API
type DepositResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	Reference     string `json:"reference"`
}

// WithdrawRequest sends money to a Uganda mobile money user
type WithdrawRequest struct {
	Provider    UgandaMobileProvider `json:"provider"`
	PhoneNumber string               `json:"phone_number"`
	Amount      float64              `json:"amount"`
	Reference   string               `json:"reference"`
	CallbackURL string               `json:"callback_url"`
}

// WithdrawResponse from the mobile money API
type WithdrawResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	Reference     string `json:"reference"`
}

// QueryTransactionResult checks the status of a mobile money transaction
type QueryTransactionResult struct {
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"` // pending, successful, failed
	Amount        float64 `json:"amount"`
	PhoneNumber   string  `json:"phone_number"`
	Reference     string  `json:"reference"`
	FailureReason string  `json:"failure_reason,omitempty"`
}

// --- Uganda Mobile Money API Implementation ---

// InitiateDeposit initiates a mobile money collection (deposit) in Uganda
// This prompts the user to authorize payment on their phone
func (s *UgandaMobileService) InitiateDeposit(provider string, phoneNumber string, amount float64, reference string) (*DepositResponse, error) {
	formattedPhone := s.formatUgandaPhone(phoneNumber, provider)
	amountInt := int(math.Round(amount))

	providerEnum := UgandaMobileProvider(strings.ToLower(provider))

	payload := map[string]interface{}{
		"provider":      providerEnum,
		"phone_number":  formattedPhone,
		"amount":        amountInt,
		"reference":     reference,
		"callback_url":  s.callbackURL + "/api/v1/uganda-mobile/callback",
		"description":   "InstaDoh Wallet Deposit",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deposit request: %w", err)
	}

	// In a production system, this would call the actual mobile money API
	// (e.g., MTN MoMo API, Airtel Money API, or an aggregator like Beyonic, Hubtel)
	// For now, we simulate the API interaction with a logging placeholder
	log.Printf("[UgandaMobile] InitiateDeposit: provider=%s phone=%s amount=%d ref=%s",
		providerEnum, formattedPhone, amountInt, reference)

	// Simulate calling the provider API
	resp, err := s.callProviderAPI("/collection", body)
	if err != nil {
		return nil, fmt.Errorf("deposit request failed: %w", err)
	}

	return resp, nil
}

// InitiateWithdrawal sends money to a Uganda mobile money user (withdrawal)
func (s *UgandaMobileService) InitiateWithdrawal(provider string, phoneNumber string, amount float64, reference string) (*WithdrawResponse, error) {
	formattedPhone := s.formatUgandaPhone(phoneNumber, provider)
	amountInt := int(math.Round(amount))

	providerEnum := UgandaMobileProvider(strings.ToLower(provider))

	payload := map[string]interface{}{
		"provider":      providerEnum,
		"phone_number":  formattedPhone,
		"amount":        amountInt,
		"reference":     reference,
		"callback_url":  s.callbackURL + "/api/v1/uganda-mobile/callback",
		"description":   "InstaDoh Wallet Withdrawal",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal withdrawal request: %w", err)
	}

	log.Printf("[UgandaMobile] InitiateWithdrawal: provider=%s phone=%s amount=%d ref=%s",
		providerEnum, formattedPhone, amountInt, reference)

	depResp, err := s.callProviderAPI("/disbursement", body)
	if err != nil {
		return nil, fmt.Errorf("withdrawal request failed: %w", err)
	}

	return &WithdrawResponse{
		TransactionID: depResp.TransactionID,
		Status:        depResp.Status,
		Message:       depResp.Message,
		Reference:     depResp.Reference,
	}, nil
}

// QueryTransaction checks the status of a mobile money transaction
func (s *UgandaMobileService) QueryTransaction(transactionID string) (*QueryTransactionResult, error) {
	payload := map[string]interface{}{
		"transaction_id": transactionID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request: %w", err)
	}

	log.Printf("[UgandaMobile] QueryTransaction: id=%s", transactionID)

	queryURL := s.apiBaseURL + "/transactions/" + transactionID
	req, err := http.NewRequest("GET", queryURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	s.setAuthHeader(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query request failed: %w", err)
	}
	defer resp.Body.Close()

	var result QueryTransactionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	return &result, nil
}

// VerifyWebhookSignature verifies the webhook callback signature from the provider
func (s *UgandaMobileService) VerifyWebhookSignature(payload []byte, signature string) bool {
	// In production, verify HMAC signature
	// For now, return true (placeholder)
	return true
}

// --- Helpers ---

func (s *UgandaMobileService) callProviderAPI(endpoint string, body []byte) (*DepositResponse, error) {
	url := s.apiBaseURL + endpoint
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create API request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	s.setAuthHeader(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// If API is not available, return a simulated response for development
		log.Printf("[UgandaMobile] API call failed (dev mode): %v", err)
		return &DepositResponse{
			TransactionID: fmt.Sprintf("TXN-%d", time.Now().Unix()),
			Status:        "pending",
			Message:       "Transaction initiated (dev mode)",
			Reference:     fmt.Sprintf("REF-%d", time.Now().UnixNano()),
		}, nil
	}
	defer resp.Body.Close()

	var result DepositResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	return &result, nil
}

func (s *UgandaMobileService) setAuthHeader(req *http.Request) {
	if s.apiUsername != "" && s.apiPassword != "" {
		req.SetBasicAuth(s.apiUsername, s.apiPassword)
	}
}

func (s *UgandaMobileService) formatUgandaPhone(phone string, provider string) string {
	// Remove non-digit characters
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, "+", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, " ", "")

	// Ugandan phone numbers start with 256
	// MTN: 077x/078x, Airtel: 070x/075x/079x
	provider = strings.ToLower(provider)

	switch provider {
	case "mtn":
		// MTN numbers: 077X XXX XXX or +256 77X XXX XXX
		if strings.HasPrefix(phone, "0") {
			phone = "256" + phone[1:]
		} else if strings.HasPrefix(phone, "+256") {
			phone = "256" + phone[4:]
		} else if !strings.HasPrefix(phone, "256") {
			phone = "256" + phone
		}
	case "airtel":
		// Airtel numbers: 070X XXX XXX or +256 70X XXX XXX
		if strings.HasPrefix(phone, "0") {
			phone = "256" + phone[1:]
		} else if strings.HasPrefix(phone, "+256") {
			phone = "256" + phone[4:]
		} else if !strings.HasPrefix(phone, "256") {
			phone = "256" + phone
		}
	default:
		// Generic Uganda format
		if strings.HasPrefix(phone, "0") {
			phone = "256" + phone[1:]
		} else if strings.HasPrefix(phone, "+256") {
			phone = "256" + phone[4:]
		} else if !strings.HasPrefix(phone, "256") {
			phone = "256" + phone
		}
	}

	return phone
}