package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"instadoh-backend/config"
)

// LightningAddressService resolves LUD-16 Lightning Addresses
// (alice@example.com) into payable BOLT11 invoices. This makes the platform
// globally interoperable with any Lightning wallet — no mobile money or phone
// number required for international sending and receiving.
type LightningAddressService struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewLightningAddressService creates the Lightning Address resolver.
func NewLightningAddressService(cfg *config.Config) *LightningAddressService {
	return &LightningAddressService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// LNURLPResponse is the standard lnurl-pay (LUD-16) metadata document.
type LNURLPResponse struct {
	Callback        string `json:"callback"`
	MaxSendable     int64  `json:"maxSendable"`
	MinSendable     int64  `json:"minSendable"`
	Metadata        string `json:"metadata"`
	CommentAllowed  int64  `json:"commentAllowed"`
	Tag             string `json:"tag"`
	AllowsNostr     bool   `json:"allowsNostr,omitempty"`
	NostrPubkey     string `json:"nostrPubkey,omitempty"`
}

// LNURLPCallbackResult is the document returned by the lnurl-pay callback.
type LNURLPCallbackResult struct {
	PR            string `json:"pr"`
	Routes        []any  `json:"routes"`
	SuccessAction *struct {
		Tag       string `json:"tag"`
		Message   string `json:"message"`
	} `json:"successAction,omitempty"`
}

// ParseLightningAddress splits "alice@example.com" into user and host parts.
func ParseLightningAddress(address string) (user, host string, err error) {
	if address == "" || !strings.Contains(address, "@") {
		return "", "", fmt.Errorf("invalid lightning address: %q", address)
	}
	parts := strings.SplitN(address, "@", 2)
	user = strings.TrimSpace(parts[0])
	host = strings.TrimSpace(parts[1])
	if user == "" || host == "" {
		return "", "", fmt.Errorf("invalid lightning address: %q", address)
	}
	return user, host, nil
}

// IsBolt11Invoice returns true if the string is a BOLT11 payment request.
func IsBolt11Invoice(destination string) bool {
	d := strings.ToLower(strings.TrimSpace(destination))
	return strings.HasPrefix(d, "lnbc") ||
		strings.HasPrefix(d, "lntb") ||
		strings.HasPrefix(d, "lntbs") ||
		strings.HasPrefix(d, "lnbcrt")
}

// IsLightningAddress returns true if the string looks like a Lightning Address.
func IsLightningAddress(destination string) bool {
	if IsBolt11Invoice(destination) {
		return false
	}
	_, _, err := ParseLightningAddress(destination)
	return err == nil
}

// IsInternalAddress reports whether the address host is InstaDoh's own domain,
// in which case the payment is settled internally rather than on-chain.
func (s *LightningAddressService) IsInternalAddress(address string) bool {
	_, host, err := ParseLightningAddress(address)
	if err != nil {
		return false
	}
	return strings.EqualFold(host, s.cfg.App.LNURLDomain)
}

// FetchLnurlInvoice resolves a Lightning Address to a BOLT11 invoice for the
// given amount in millisatoshis.
func (s *LightningAddressService) FetchLnurlInvoice(address string, amountMsat int64, comment string) (string, error) {
	user, host, err := ParseLightningAddress(address)
	if err != nil {
		return "", err
	}

	lnurlpURL := fmt.Sprintf("https://%s/.well-known/lnurlp/%s", host, user)
	httpResp, err := s.httpClient.Get(lnurlpURL)
	if err != nil {
		return "", fmt.Errorf("failed to resolve lightning address %s: %w", address, err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lightning address %s not found (HTTP %d)", address, httpResp.StatusCode)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read lnurl-pay response: %w", err)
	}

	var meta LNURLPResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return "", fmt.Errorf("invalid lnurl-pay response from %s: %w", address, err)
	}
	if meta.Callback == "" || meta.Tag != "payRequest" {
		return "", fmt.Errorf("unsupported lnurl-pay response from %s", address)
	}
	if amountMsat > meta.MaxSendable {
		return "", fmt.Errorf("amount %d msat exceeds max %d msat for %s", amountMsat, meta.MaxSendable, address)
	}
	if amountMsat < meta.MinSendable {
		return "", fmt.Errorf("amount %d msat below min %d msat for %s", amountMsat, meta.MinSendable, address)
	}

	callback, err := url.Parse(meta.Callback)
	if err != nil {
		return "", fmt.Errorf("invalid callback URL: %w", err)
	}
	q := callback.Query()
	q.Set("amount", fmt.Sprintf("%d", amountMsat))
	if comment != "" && meta.CommentAllowed > 0 {
		if int64(len(comment)) > meta.CommentAllowed {
			comment = comment[:meta.CommentAllowed]
		}
		q.Set("comment", comment)
	}
	callback.RawQuery = q.Encode()

	invResp, err := s.httpClient.Get(callback.String())
	if err != nil {
		return "", fmt.Errorf("failed to request invoice from %s: %w", address, err)
	}
	defer invResp.Body.Close()

	invBody, err := io.ReadAll(invResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read invoice response: %w", err)
	}

	var result LNURLPCallbackResult
	if err := json.Unmarshal(invBody, &result); err != nil {
		return "", fmt.Errorf("invalid invoice response: %w", err)
	}
	if result.PR == "" {
		return "", fmt.Errorf("no invoice returned for %s", address)
	}

	return result.PR, nil
}