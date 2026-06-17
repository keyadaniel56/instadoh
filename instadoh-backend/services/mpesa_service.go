package services

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// MpesaService handles Safaricom Daraja API integration for M-Pesa
// This enables Kenyan users to deposit (STK Push) and withdraw (B2C) via M-Pesa
type MpesaService struct {
	consumerKey        string
	consumerSecret     string
	passkey            string
	shortCode          string
	environment        string // "sandbox" or "production"
	callbackURL        string
	httpClient         *http.Client
	accessToken        string
	accessTokenExpires time.Time
}

// MpesaConfig holds Daraja API configuration
type MpesaConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	Passkey        string
	ShortCode      string
	Environment    string
	CallbackURL    string
}

// NewMpesaService creates a new Daraja API client
func NewMpesaService(cfg *MpesaConfig) *MpesaService {
	return &MpesaService{
		consumerKey:    cfg.ConsumerKey,
		consumerSecret: cfg.ConsumerSecret,
		passkey:        cfg.Passkey,
		shortCode:      cfg.ShortCode,
		environment:    cfg.Environment,
		callbackURL:    cfg.CallbackURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- OAuth Token ---

func (s *MpesaService) getAccessToken() (string, error) {
	now := time.Now()
	if s.accessToken != "" && now.Before(s.accessTokenExpires) {
		return s.accessToken, nil
	}

	tokenURL := s.baseURL() + "/oauth/v1/generate?grant_type=client_credentials"
	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.SetBasicAuth(s.consumerKey, s.consumerSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch access token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	s.accessToken = result.AccessToken
	expiresIn, _ := strconv.Atoi(result.ExpiresIn)
	if expiresIn < 60 {
		expiresIn = 60
	}
	s.accessTokenExpires = now.Add(time.Duration(expiresIn-10) * time.Second)

	return s.accessToken, nil
}

// --- STK Push (Customer-to-Business) for deposits ---

// STKPush initiates an M-Pesa STK Push prompt on the customer's phone
// The user enters their M-Pesa PIN on their phone to authorize the payment
// This is used for deposits (funding their InstaDoh wallet)
type STKPushResponse struct {
	CheckoutRequestID string `json:"CheckoutRequestID"`
	MerchantRequestID string `json:"MerchantRequestID"`
	ResponseCode      string `json:"ResponseCode"`
	ResponseDesc      string `json:"ResponseDesc"`
	CustomerMessage   string `json:"CustomerMessage"`
}

// InitiateSTKPush sends an STK Push request to the Daraja API
func (s *MpesaService) InitiateSTKPush(phoneNumber string, amount float64, accountRef, transactionDesc string) (*STKPushResponse, error) {
	token, err := s.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	timestamp := s.generateTimestamp()
	password := s.generatePassword(timestamp)

	// Format phone: remove leading 0 or +254, ensure 254 format
	formattedPhone := s.formatPhoneNumber(phoneNumber)

	// Round amount to whole number (KES has no cents in mobile money)
	amountInt := int(math.Round(amount))

	payload := map[string]interface{}{
		"BusinessShortCode": s.shortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
		"Amount":            amountInt,
		"PartyA":            formattedPhone,
		"PartyB":            s.shortCode,
		"PhoneNumber":       formattedPhone,
		"CallBackURL":       s.callbackURL + "/api/v1/mpesa/callback",
		"AccountReference":  accountRef,
		"TransactionDesc":   transactionDesc,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal STK push request: %w", err)
	}

	stkURL := s.baseURL() + "/mpesa/stkpush/v1/processrequest"
	req, err := http.NewRequest("POST", stkURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create STK push request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("STK push request failed: %w", err)
	}
	defer resp.Body.Close()

	var result STKPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode STK push response: %w", err)
	}

	return &result, nil
}

// QuerySTKPushStatus checks the status of an STK Push transaction
func (s *MpesaService) QuerySTKPushStatus(checkoutRequestID string) (*STKPushQueryResponse, error) {
	token, err := s.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	timestamp := s.generateTimestamp()
	password := s.generatePassword(timestamp)

	payload := map[string]interface{}{
		"BusinessShortCode": s.shortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"CheckoutRequestID": checkoutRequestID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request: %w", err)
	}

	queryURL := s.baseURL() + "/mpesa/stkpushquery/v1/query"
	req, err := http.NewRequest("POST", queryURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create query request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query request failed: %w", err)
	}
	defer resp.Body.Close()

	var result STKPushQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	return &result, nil
}

// STKPushQueryResponse holds the result of an STK Push status query
type STKPushQueryResponse struct {
	ResponseCode       string `json:"ResponseCode"`
	ResponseDesc       string `json:"ResponseDesc"`
	MerchantRequestID  string `json:"MerchantRequestID"`
	CheckoutRequestID  string `json:"CheckoutRequestID"`
	ResultCode         string `json:"ResultCode"`
	ResultDesc         string `json:"ResultDesc"`
}

// --- B2C (Business-to-Consumer) for withdrawals ---

// B2CRequest sends money from the business to a customer's M-Pesa account
// Used when a Kenyan user wants to withdraw from their InstaDoh wallet
type B2CResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDesc             string `json:"ResponseDesc"`
}

// InitiateB2CPayment initiates a B2C (Business to Customer) payment
func (s *MpesaService) InitiateB2CPayment(phoneNumber string, amount float64, remarks string) (*B2CResponse, error) {
	token, err := s.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	formattedPhone := s.formatPhoneNumber(phoneNumber)
	amountInt := int(math.Round(amount))

	// Use the security credential for B2C API
	securityCredential, err := s.getSecurityCredential()
	if err != nil {
		log.Printf("Warning: Could not generate security credential, using passkey: %v", err)
		securityCredential = s.passkey
	}

	payload := map[string]interface{}{
		"InitiatorName":      "testapi",
		"SecurityCredential": securityCredential,
		"CommandID":          "BusinessPayment",
		"Amount":             amountInt,
		"PartyA":             s.shortCode,
		"PartyB":             formattedPhone,
		"Remarks":            remarks,
		"QueueTimeOutURL":    s.callbackURL + "/api/v1/mpesa/timeout",
		"ResultURL":          s.callbackURL + "/api/v1/mpesa/result",
		"Occasion":           "InstaDoh Withdrawal",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal B2C request: %w", err)
	}

	b2cURL := s.baseURL() + "/mpesa/b2c/v1/paymentrequest"
	req, err := http.NewRequest("POST", b2cURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create B2C request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("B2C request failed: %w", err)
	}
	defer resp.Body.Close()

	var result B2CResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode B2C response: %w", err)
	}

	return &result, nil
}

// --- STK Push Callback Processing ---

// STKCallbackData processes the callback from Safaricom after STK Push
type STKCallbackData struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string      `json:"Name"`
					Value interface{} `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

// ProcessSTKCallbackData processes STK Push callback data and returns checkoutRequestID, amount, error
func ProcessSTKCallbackData(callbackData *STKCallbackData) (string, float64, error) {
	cb := callbackData.Body.StkCallback
	checkoutRequestID := cb.CheckoutRequestID

	if cb.ResultCode != 0 {
		return checkoutRequestID, 0, fmt.Errorf("STK push failed with code %d: %s", cb.ResultCode, cb.ResultDesc)
	}

	// Extract amount from callback metadata
	var amount float64
	for _, item := range cb.CallbackMetadata.Item {
		switch item.Name {
		case "Amount":
			if v, ok := item.Value.(float64); ok {
				amount = v
			}
		case "PhoneNumber":
			// Used for logging/verification
		case "MpesaReceiptNumber":
			// M-Pesa transaction ID for reconciliation
		}
	}

	return checkoutRequestID, amount, nil
}

// --- Helper functions ---

func (s *MpesaService) baseURL() string {
	if s.environment == "production" {
		return "https://api.safaricom.co.ke"
	}
	return "https://sandbox.safaricom.co.ke"
}

func (s *MpesaService) generateTimestamp() string {
	now := time.Now()
	return now.Format("20060102150405")
}

func (s *MpesaService) generatePassword(timestamp string) string {
	// Password is base64(Shortcode + Passkey + Timestamp)
	data := s.shortCode + s.passkey + timestamp
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func (s *MpesaService) formatPhoneNumber(phone string) string {
	// Remove any non-digit characters
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, "+", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, " ", "")

	// If starts with 0, replace with 254
	if strings.HasPrefix(phone, "0") {
		phone = "254" + phone[1:]
	}
	// If starts with +254, remove the +
	if strings.HasPrefix(phone, "+254") {
		phone = "254" + phone[4:]
	}
	// If doesn't start with 254, assume Kenyan number and add prefix
	if !strings.HasPrefix(phone, "254") {
		phone = "254" + phone
	}

	return phone
}

func (s *MpesaService) getSecurityCredential() (string, error) {
	// In production, this should use the actual certificate from Safaricom
	// For now, return the passkey as a fallback
	// Read the certificate file if it exists
	certPath := os.Getenv("MPESA_CERT_PATH")
	if certPath == "" {
		return s.passkey, nil
	}

	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("failed to read cert: %w", err)
	}

	// Parse the certificate
	block, _ := x509.ParseCertificate(certBytes)
	if block == nil {
		// Try PEM decode
		return "", fmt.Errorf("failed to parse certificate")
	}

	pubKey := block.PublicKey.(*rsa.PublicKey)
	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(s.passkey), nil)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}