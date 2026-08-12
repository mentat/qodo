package services

import (
	"context"
	"fmt"
	"sync"
)

// ERPClient books an approved invoice into the customer's ERP. The ERP stays
// the system of record — nothing is released for payment until it has a
// document ID here.
type ERPClient interface {
	// PostInvoice books the invoice and returns the ERP document ID.
	PostInvoice(ctx context.Context, inv Invoice) (string, error)
}

// PaymentRequest is a single payment instruction handed to the gateway.
type PaymentRequest struct {
	// IdempotencyKey lets the gateway collapse duplicate submissions of the
	// same logical payment.
	IdempotencyKey string
	InvoiceID      string
	EntityID       string
	AmountCents    int64
	CurrencyCode   string
	ERPDocumentID  string
}

// PaymentGateway releases funds for an approved, ERP-booked invoice.
type PaymentGateway interface {
	// Submit returns the gateway's payment reference.
	Submit(ctx context.Context, req PaymentRequest) (string, error)
}

// StubERPClient stands in for a real ERP connector in local dev and demos.
// It fabricates a stable document ID from the invoice.
type StubERPClient struct{}

// PostInvoice implements ERPClient.
func (StubERPClient) PostInvoice(_ context.Context, inv Invoice) (string, error) {
	return fmt.Sprintf("ERP-%s-%s", inv.EntityID, inv.ID), nil
}

// StubPaymentGateway stands in for a real payment rail. It records the
// idempotency keys it has seen so local runs behave like a gateway that
// deduplicates, and is safe for concurrent use.
type StubPaymentGateway struct {
	mu   sync.Mutex
	seen map[string]string
}

// Submit implements PaymentGateway. A key it has already settled returns the
// original reference rather than paying twice.
func (g *StubPaymentGateway) Submit(_ context.Context, req PaymentRequest) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.seen == nil {
		g.seen = map[string]string{}
	}
	if ref, ok := g.seen[req.IdempotencyKey]; ok {
		return ref, nil
	}
	ref := fmt.Sprintf("PMT-%s-%d", req.InvoiceID, len(g.seen)+1)
	g.seen[req.IdempotencyKey] = ref
	return ref, nil
}

// SettledCount reports how many distinct payments the stub has released.
// Useful for asserting that a retry didn't pay twice.
func (g *StubPaymentGateway) SettledCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.seen)
}
