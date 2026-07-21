// Package payment provides Midtrans payment gateway integration for GuestFlow.
package payment

import (
	"crypto/sha512"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	midtrans "github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

// Client wraps the Midtrans Snap API client.
type Client struct {
	serverKey    string
	clientKey    string
	isProduction bool
	snapClient   snap.Client
}

// NewClient creates a new Midtrans payment client.
func NewClient(serverKey, clientKey string, isProduction bool) *Client {
	env := midtrans.Sandbox
	if isProduction {
		env = midtrans.Production
	}

	var sc snap.Client
	sc.New(serverKey, env)

	return &Client{
		serverKey:    serverKey,
		clientKey:    clientKey,
		isProduction: isProduction,
		snapClient:   sc,
	}
}

// ClientKey returns the Midtrans client key (for frontend Snap.js).
func (c *Client) ClientKey() string {
	return c.clientKey
}

// SnapURL returns the Snap.js CDN URL based on environment.
func (c *Client) SnapURL() string {
	if c.isProduction {
		return "https://app.midtrans.com/snap/snap.js"
	}
	return "https://app.sandbox.midtrans.com/snap/snap.js"
}

// CreateOrderID generates a unique Midtrans order ID.
// Format: GF-<tenantID_short>-<timestamp>-<random>
func CreateOrderID(tenantID uuid.UUID) string {
	short := strings.ReplaceAll(tenantID.String(), "-", "")[:8]
	ts := time.Now().Unix()
	rnd := uuid.New().String()[:4]
	return fmt.Sprintf("GF-%s-%d-%s", short, ts, rnd)
}

// CreateSnapRequest holds parameters for creating a Midtrans Snap transaction.
type CreateSnapRequest struct {
	OrderID     string
	AmountIDR   int64
	PlanName    string
	BillingCycle string
	TenantID    uuid.UUID
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
}

// CreateTransaction creates a Snap transaction and returns the snap token.
func (c *Client) CreateTransaction(req CreateSnapRequest) (string, error) {
	billingLabel := "bulanan"
	if req.BillingCycle == "yearly" {
		billingLabel = "tahunan"
	}

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  req.OrderID,
			GrossAmt: req.AmountIDR,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: req.CustomerName,
			Email: req.CustomerEmail,
			Phone: req.CustomerPhone,
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    req.OrderID,
				Name:  fmt.Sprintf("GuestFlow %s (%s)", req.PlanName, billingLabel),
				Price: req.AmountIDR,
				Qty:   1,
			},
		},
		Expiry: &snap.ExpiryDetails{
			Unit:     "day",
			Duration: 1,
		},
	}

	snapResp, err := c.snapClient.CreateTransaction(snapReq)
	if err != nil {
		return "", fmt.Errorf("midtrans create transaction: %w", err)
	}

	return snapResp.Token, nil
}

// NotificationPayload represents the Midtrans payment notification body.
type NotificationPayload struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
	Currency          string `json:"currency"`
	VANumbers         []struct {
		Bank     string `json:"bank"`
		VANumber string `json:"va_number"`
	} `json:"va_numbers"`
}

// VerifySignature validates the notification payload using SHA512.
// Midtrans signature: SHA512(order_id + status_code + gross_amount + server_key)
func (c *Client) VerifySignature(orderID, statusCode, grossAmount, signatureKey string) bool {
	raw := orderID + statusCode + grossAmount + c.serverKey
	hash := sha512.Sum512([]byte(raw))
	expected := fmt.Sprintf("%x", hash)
	return strings.EqualFold(expected, signatureKey)
}

// IsSuccessStatus returns true if the transaction status indicates a successful payment.
func IsSuccessStatus(status, fraudStatus string) bool {
	switch status {
	case "capture":
		return fraudStatus == "accept"
	case "settlement":
		return true
	default:
		return false
	}
}

// IsFailedStatus returns true if the transaction has failed or been cancelled.
func IsFailedStatus(status string) bool {
	return status == "deny" || status == "cancel" || status == "failure"
}

// IsExpiredStatus returns true if the transaction has expired.
func IsExpiredStatus(status string) bool {
	return status == "expire"
}
