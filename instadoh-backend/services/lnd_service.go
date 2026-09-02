package services

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"instadoh-backend/config"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// LNDService handles all Lightning Network interactions via a modern LND gRPC
// client. A single hot-wallet LND node sits behind the service and is shared by
// all users. Lightning is used purely as an instant settlement rail: users see
// their balances in local currency and are not exposed to the underlying BTC.
type LNDService struct {
	client      lnrpc.LightningClient
	cfg         *config.LNDConfig
	isConnected bool

	mu          sync.Mutex
	subscribers []func(*lnrpc.Invoice)
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

// connect dials LND and authenticates using the admin macaroon and TLS cert.
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
		grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)
			return streamer(ctx, desc, cc, method, opts...)
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

// CreateInvoice generates a real Lightning (bolt11) invoice for a given amount
// in millisatoshis. The resulting payment request string can be shared directly
// and scanned by any Lightning wallet.
func (s *LNDService) CreateInvoice(amountMsat int64, description string, expirySeconds int64) (*lnrpc.AddInvoiceResponse, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	expiry := expirySeconds
	if expiry <= 0 {
		expiry = 3600
	}

	invoice := &lnrpc.Invoice{
		Memo:      description,
		ValueMsat: amountMsat,
		Expiry:    expiry,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.client.AddInvoice(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return resp, nil
}

// PayInvoice pays a Lightning (bolt11) invoice via the synchronous SendPayment
// API. The specified amount is in millisatoshis (converted from the user's
// local currency by the caller).
func (s *LNDService) PayInvoice(paymentRequest string, amountMsat int64) (*SendPaymentResult, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	// Prefer paying with the payment request string. LND will validate the
	// amount against the invoice (for fixed-amount invoices) and route it.
	req := &lnrpc.SendRequest{
		PaymentRequest: paymentRequest,
		AmtMsat:        amountMsat,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := s.client.SendPaymentSync(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("payment RPC failed: %w", err)
	}

	if resp.PaymentError != "" {
		return nil, fmt.Errorf("payment rejected by LND: %s", resp.PaymentError)
	}

	return &SendPaymentResult{
		PaymentHash: resp.PaymentHash,
		Preimage:    resp.PaymentPreimage,
		FeeMsat:     routeFeeMsat(resp.PaymentRoute),
	}, nil
}

// SendPaymentResult holds the result of a payment
type SendPaymentResult struct {
	PaymentHash []byte
	Preimage    []byte
	FeeMsat     int64
}

// routeFeeMsat sums the routing fees across all hops of a settled route.
func routeFeeMsat(route *lnrpc.Route) int64 {
	if route == nil {
		return 0
	}
	var total int64
	for _, hop := range rangeHops(route) {
		total += hop.FeeMsat
	}
	return total
}

// rangeHops safely iterates over the hops in a route (avoids nil receivers).
func rangeHops(route *lnrpc.Route) []*lnrpc.Hop {
	if route == nil {
		return nil
	}
	return route.Hops
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
// using the real DecodePayReq gRPC method.
func (s *LNDService) DecodePaymentRequest(payReq string) (*DecodedPaymentRequest, error) {
	if !s.isConnected {
		return nil, fmt.Errorf("LND not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	decoded, err := s.client.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: payReq})
	if err != nil {
		return nil, fmt.Errorf("failed to decode payment request: %w", err)
	}

	return &DecodedPaymentRequest{
		Destination: decoded.Destination,
		PaymentHash: decoded.PaymentHash,
		Description: decoded.Description,
		AmountMsat:  decoded.NumMsat,
		NumSatoshis: decoded.NumSatoshis,
		Expiry:      decoded.Expiry,
		Timestamp:   decoded.Timestamp,
	}, nil
}

// DecodedPaymentRequest holds parsed Bolt11 invoice data
type DecodedPaymentRequest struct {
	Destination string
	PaymentHash string
	Description string
	AmountMsat  int64
	NumSatoshis int64
	Expiry      int64
	Timestamp   int64
}

// IsSettled reports whether a fetched invoice has reached the SETTLED state.
func IsInvoiceSettled(invoice *lnrpc.Invoice) bool {
	if invoice == nil {
		return false
	}
	return invoice.GetState() == lnrpc.Invoice_SETTLED
}

// SubscribeInvoiceSettlements opens a persistent stream of settled invoices and
// invokes the provided callback for each. This is how the system lears that money
// has arrived and immediately credits the user's local-currency balance.
func (s *LNDService) SubscribeInvoiceSettlements(callback func(*lnrpc.Invoice)) {
	s.mu.Lock()
	s.subscribers = append(s.subscribers, callback)
	s.mu.Unlock()

	if !s.isConnected {
		return
	}

	go func() {
		for {
			if !s.isConnected {
				time.Sleep(5 * time.Second)
				continue
			}

			ctx, cancel := context.WithCancel(context.Background())
			stream, err := s.client.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{})
			if err != nil {
				cancel()
				log.Printf("WARNING: failed to subscribe to invoices, retrying: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for {
				invoice, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("WARNING: invoice stream error: %v", err)
					break
				}
				if invoice == nil {
					continue
				}
				if IsInvoiceSettled(invoice) {
					s.notify(invoice)
				}
			}
			cancel()
			log.Println("Invoice subscription stream ended, reconnecting...")
			time.Sleep(3 * time.Second)
		}
	}()
}

func (s *LNDService) notify(invoice *lnrpc.Invoice) {
	s.mu.Lock()
	subs := make([]func(*lnrpc.Invoice), len(s.subscribers))
	copy(subs, s.subscribers)
	s.mu.Unlock()

	for _, sub := range subs {
		go sub(invoice)
	}
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
// exchangeRate is "how many units of local currency per 1 BTC-equivalent USD",
// however in practice the caller passes the BTC price in the same units.
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
