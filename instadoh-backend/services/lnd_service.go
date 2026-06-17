package services

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"instadoh-backend/config"
	"instadoh-backend/models"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// LNDService handles all Lightning Network interactions
type LNDService struct {
	client      lnrpc.LightningClient
	cfg         *config.LNDConfig
	isConnected bool
}

// NewLNDService creates a new LND gRPC client
func NewLNDService(cfg *config.LNDConfig) *LNDService {
	svc := &LNDService{
		cfg: cfg,
	}

	if err := svc.connect(); err != nil {
		log.Printf("WARNING: Failed to connect to LND: %v. Running in offline mode.", err)
		svc.isConnected = false
		return svc
	}

	svc.isConnected = true
	log.Println("LND service connected successfully")
	return svc
}

func (s *LNDService) connect() error {
	macaroonBytes, err := loadMacaroonBytes(s.cfg.MacaroonPath)
	if err != nil {
		return fmt.Errorf("failed to load macaroon: %w", err)
	}

	creds, err := loadTLSCredentials(s.cfg.TLSCertPath)
	if err != nil {
		return fmt.Errorf("failed to load TLS cert: %w", err)
	}

	// Build gRPC dial options
	macaroonHex := hex.EncodeToString(macaroonBytes)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50 * 1024 * 1024)),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			// Add macaroon as gRPC metadata (required by LND for authentication)
			ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)
			return invoker(ctx, method, req, reply, cc, opts...)
		}),
	}

	conn, err := grpc.Dial(s.cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial LND: %w", err)
	}

	s.client = lnrpc.NewLightningClient(conn)

	return nil
}

// IsConnected returns the connection status to LND
func (s *LNDService) IsConnected() bool {
	return s.isConnected
}

// CreateInvoice generates a Lightning invoice for a given amount in millisatoshis
// Note: Uses the lnrpc v0.0.2 API where Invoice.Value is in satoshis
func (s *LNDService) CreateInvoice(user *models.User, amountMsat int64, description string, expirySeconds int64) (*lnrpc.AddInvoiceResponse, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	// Convert msat to satoshis (truncate, rounding up for safety)
	valueSat := int64(math.Ceil(float64(amountMsat) / 1000.0))

	invoice := &lnrpc.Invoice{
		Memo:    description,
		Value:   valueSat,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.client.AddInvoice(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return resp, nil
}

// PayInvoice pays a Lightning invoice using the streaming SendPayment API
// Note: The v0.0.2 lnrpc API does not support paying bolt11 invoices directly via PaymentRequest.
// This implementation uses the SendPayment streaming method with basic payment fields.
func (s *LNDService) PayInvoice(paymentRequest string, amountMsat int64) (*SendPaymentResult, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	// Convert msat to satoshis
	valueSat := int64(math.Ceil(float64(amountMsat) / 1000.0))

	// Use the low-level SendPayment streaming API
	// SendRequest only supports Dest, Amt, PaymentHash, FastSend in v0.0.2
	// We attempt to decode the payment request to extract destination and payment hash
	hashBytes, destBytes, err := parseBolt11(paymentRequest)
	if err != nil {
		// If we can't parse the bolt11, we can't pay it with this old API
		return nil, fmt.Errorf("cannot pay bolt11 invoice with this LND API version (v0.0.2): %w", err)
	}

	req := &lnrpc.SendRequest{
		Dest:        destBytes,
		Amt:         valueSat,
		PaymentHash: hashBytes,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	stream, err := s.client.SendPayment(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate payment stream: %w", err)
	}

	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("failed to send payment request: %w", err)
	}

	_, err = stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	return &SendPaymentResult{
		PaymentHash: hashBytes,
	}, nil
}

// SendPaymentResult holds the result of a payment
type SendPaymentResult struct {
	PaymentHash []byte
}

// GetInvoiceStatus checks the status of an invoice
func (s *LNDService) GetInvoiceStatus(paymentHash string) (*lnrpc.Invoice, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	hashBytes, err := hex.DecodeString(paymentHash)
	if err != nil {
		return nil, fmt.Errorf("invalid payment hash: %w", err)
	}

	req := &lnrpc.PaymentHash{
		RHash: hashBytes,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	invoice, err := s.client.LookupInvoice(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup invoice: %w", err)
	}

	return invoice, nil
}

// DecodePaymentRequest decodes a Lightning payment request (Bolt11 invoice)
// Uses basic parsing since lnrpc v0.0.2 doesn't have DecodePayReq
func (s *LNDService) DecodePaymentRequest(payReq string) (*DecodedPaymentRequest, error) {
	hashBytes, destBytes, err := parseBolt11(payReq)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payment request: %w", err)
	}

	dest := ""
	if len(destBytes) > 0 {
		dest = hex.EncodeToString(destBytes)
	}

	return &DecodedPaymentRequest{
		Destination:  dest,
		PaymentHash:  hex.EncodeToString(hashBytes),
		Description:  "",
	}, nil
}

// DecodedPaymentRequest holds parsed Bolt11 invoice data
type DecodedPaymentRequest struct {
	Destination  string
	PaymentHash  string
	Description  string
}

// GetNodeInfo returns information about the LND node
func (s *LNDService) GetNodeInfo() (*lnrpc.GetInfoResponse, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := s.client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	return info, nil
}

// GetChannelBalance returns the total channel balance
func (s *LNDService) GetChannelBalance() (*lnrpc.ChannelBalanceResponse, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := s.client.ChannelBalance(ctx, &lnrpc.ChannelBalanceRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get channel balance: %w", err)
	}

	return balance, nil
}

// FiatToBTC converts a fiat amount to millisatoshis (msat)
func FiatToBTC(amount float64, exchangeRate float64) int64 {
	btcAmount := amount / exchangeRate
	msat := btcAmount * 1e11
	return int64(math.Round(msat))
}

// BTCToFiat converts millisatoshis back to fiat
func BTCToFiat(msat int64, exchangeRate float64) float64 {
	btcAmount := float64(msat) / 1e11
	return btcAmount * exchangeRate
}

// --- Bolt11 parsing helpers ---
// Minimal parsing since lnrpc v0.0.2 doesn't have DecodePayReq

// parseBolt11 does minimal bolt11 invoice parsing to extract payment hash and destination.
// Returns (paymentHashBytes, destPubKeyBytes, error)
func parseBolt11(payReq string) ([]byte, []byte, error) {
	if len(payReq) < 60 {
		return nil, nil, fmt.Errorf("invalid bolt11 invoice: too short")
	}

	// Extract the payment hash from the bolt11 invoice
	// In a proper implementation this would use a bolt11 decoding library
	// For now, we return an error indicating this old LND API doesn't support paying bolt11 invoices
	return nil, nil, fmt.Errorf("bolt11 invoice payment requires a newer LND API version")
}

// --- Helper functions ---

func loadTLSCredentials(tlsCertPath string) (credentials.TransportCredentials, error) {
	if tlsCertPath == "" {
		return insecure.NewCredentials(), nil
	}

	// Try as file path first
	if _, err := os.Stat(tlsCertPath); err == nil {
		creds, err := credentials.NewClientTLSFromFile(tlsCertPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS cert from file: %w", err)
		}
		return creds, nil
	}

	// Try as raw PEM data
	cert, err := tls.X509KeyPair([]byte(tlsCertPath), []byte(tlsCertPath))
	if err != nil {
		return insecure.NewCredentials(), nil
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}

func loadMacaroonBytes(macaroonPath string) ([]byte, error) {
	if macaroonPath == "" {
		return nil, fmt.Errorf("macaroon path is empty")
	}

	// Try as hex string first
	if bytes, err := hex.DecodeString(macaroonPath); err == nil && len(bytes) > 0 {
		return bytes, nil
	}

	// Try as file path
	bytes, err := os.ReadFile(macaroonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read macaroon file: %w", err)
	}

	return bytes, nil
}