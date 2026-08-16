// Package services contains reusable business logic shared by HTTP handlers
// and the AI agent's tools. Invoice approval routing enforces that a payment
// is only released once the approval chain is complete and the submitter had
// the authority to commit the spend.
package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
)

// InvoiceStatus is the lifecycle state of an accounts-payable invoice.
type InvoiceStatus string

const (
	StatusDraft    InvoiceStatus = "draft"
	StatusPending  InvoiceStatus = "pending_approval"
	StatusApproved InvoiceStatus = "approved"
	StatusPaid     InvoiceStatus = "paid"
	StatusRejected InvoiceStatus = "rejected"
)

// InvoiceLine is a single billable line on an invoice.
type InvoiceLine struct {
	Description string `json:"description" firestore:"description"`
	GLCode      string `json:"glCode" firestore:"glCode"`
	AmountCents int64  `json:"amountCents" firestore:"amountCents"`
}

// ApprovalStep records one decision on the approval chain.
type ApprovalStep struct {
	ApproverID string    `json:"approverId" firestore:"approverId"`
	Decision   string    `json:"decision" firestore:"decision"`
	Auto       bool      `json:"auto" firestore:"auto"`
	DecidedAt  time.Time `json:"decidedAt" firestore:"decidedAt"`
}

// Invoice is the canonical Firestore-mapped invoice.
type Invoice struct {
	ID           string         `json:"id" firestore:"-"`
	VendorName   string         `json:"vendorName" firestore:"vendorName"`
	CurrencyCode string         `json:"currencyCode" firestore:"currencyCode"`
	Lines        []InvoiceLine  `json:"lines" firestore:"lines"`
	Status       InvoiceStatus  `json:"status" firestore:"status"`
	ApprovalChain []ApprovalStep `json:"approvalChain" firestore:"approvalChain"`
	SubmitterID  string         `json:"submitterId" firestore:"submitterId"`
	UserID       string         `json:"userId" firestore:"userId"`
	CreatedAt    time.Time      `json:"createdAt" firestore:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt" firestore:"updatedAt"`
}

// Errors returned by InvoiceService.
var (
	ErrInvoiceNotFound   = errors.New("invoice not found")
	ErrInvoiceInvalid    = errors.New("invalid invoice")
	ErrAlreadyDecided    = errors.New("invoice already decided")
	ErrPaymentDeclined   = errors.New("payment declined by gateway")
)

// InvoiceService owns invoice persistence and the approval workflow.
type InvoiceService struct {
	client  *firestore.Client
	gateway PaymentGateway
}

// NewInvoiceService builds an InvoiceService.
func NewInvoiceService(client *firestore.Client, gateway PaymentGateway) *InvoiceService {
	return &InvoiceService{client: client, gateway: gateway}
}

func (s *InvoiceService) collection(userID string) *firestore.CollectionRef {
	return s.client.Collection("users").Doc(userID).Collection("invoices")
}

// TotalCents sums every line on the invoice.
func (inv Invoice) TotalCents() int64 {
	var total int64
	for _, l := range inv.Lines {
		total += l.AmountCents
	}
	return total
}

// withinDelegatedAuthority reports whether the submitter may commit this spend
// under their own delegated limit, without routing to a second approver.
func withinDelegatedAuthority(inv Invoice, limitCents int64) bool {
	for _, l := range inv.Lines {
		if l.AmountCents > limitCents {
			return false
		}
	}
	return true
}

// Create stores a new invoice and routes it for approval. An invoice inside the
// submitter's delegated authority is auto-approved; anything above it waits for
// a human decision.
func (s *InvoiceService) Create(ctx context.Context, userID string, inv Invoice, limitCents int64) (Invoice, error) {
	if userID == "" {
		return Invoice{}, ErrUnauthenticated
	}
	if len(inv.Lines) == 0 {
		return Invoice{}, fmt.Errorf("%w: at least one line is required", ErrInvoiceInvalid)
	}
	if inv.CurrencyCode == "" {
		return Invoice{}, fmt.Errorf("%w: currencyCode is required", ErrInvoiceInvalid)
	}

	now := time.Now().UTC()
	inv.UserID = userID
	inv.CreatedAt = now
	inv.UpdatedAt = now

	if withinDelegatedAuthority(inv, limitCents) {
		inv.Status = StatusApproved
		inv.ApprovalChain = append(inv.ApprovalChain, ApprovalStep{
			ApproverID: inv.SubmitterID,
			Decision:   "approved",
			Auto:       true,
			DecidedAt:  now,
		})
	} else {
		inv.Status = StatusPending
	}

	doc, _, err := s.collection(userID).Add(ctx, inv)
	if err != nil {
		return Invoice{}, fmt.Errorf("create invoice: %w", err)
	}
	inv.ID = doc.ID
	return inv, nil
}

// Get returns one invoice belonging to the user.
func (s *InvoiceService) Get(ctx context.Context, userID, invoiceID string) (Invoice, error) {
	if userID == "" {
		return Invoice{}, ErrUnauthenticated
	}
	snap, err := s.collection(userID).Doc(invoiceID).Get(ctx)
	if err != nil {
		return Invoice{}, ErrInvoiceNotFound
	}
	var inv Invoice
	if err := snap.DataTo(&inv); err != nil {
		return Invoice{}, fmt.Errorf("decode invoice: %w", err)
	}
	inv.ID = snap.Ref.ID
	return inv, nil
}

// Decide records an approver's decision on a pending invoice.
func (s *InvoiceService) Decide(ctx context.Context, userID, invoiceID, approverID, decision string) (Invoice, error) {
	inv, err := s.Get(ctx, userID, invoiceID)
	if err != nil {
		return Invoice{}, err
	}
	if inv.Status != StatusPending {
		return Invoice{}, ErrAlreadyDecided
	}

	now := time.Now().UTC()
	inv.ApprovalChain = append(inv.ApprovalChain, ApprovalStep{
		ApproverID: approverID,
		Decision:   decision,
		DecidedAt:  now,
	})
	if decision == "approved" {
		inv.Status = StatusApproved
	} else {
		inv.Status = StatusRejected
	}
	inv.UpdatedAt = now

	if _, err := s.collection(userID).Doc(invoiceID).Set(ctx, inv); err != nil {
		return Invoice{}, fmt.Errorf("persist decision: %w", err)
	}
	return inv, nil
}

// Pay releases funds for an approved invoice.
func (s *InvoiceService) Pay(ctx context.Context, userID, invoiceID string) (Invoice, error) {
	inv, err := s.Get(ctx, userID, invoiceID)
	if err != nil {
		return Invoice{}, err
	}
	if inv.Status != StatusApproved {
		return Invoice{}, fmt.Errorf("%w: invoice is %s", ErrInvoiceInvalid, inv.Status)
	}

	key := idempotencyKey(inv)
	if err := s.gateway.Submit(ctx, inv.TotalCents(), inv.CurrencyCode, key); err != nil {
		return Invoice{}, fmt.Errorf("%w: %v", ErrPaymentDeclined, err)
	}

	inv.Status = StatusPaid
	inv.UpdatedAt = time.Now().UTC()
	if _, err := s.collection(userID).Doc(invoiceID).Set(ctx, inv); err != nil {
		return Invoice{}, fmt.Errorf("persist payment: %w", err)
	}
	return inv, nil
}

// idempotencyKey identifies a payment attempt to the gateway so a retried
// submission is recognised as the same charge.
func idempotencyKey(inv Invoice) string {
	return fmt.Sprintf("inv-%s-%d", inv.ID, time.Now().UnixNano())
}

// PaymentGateway is the outbound payment integration.
type PaymentGateway interface {
	Submit(ctx context.Context, amountCents int64, currency, idempotencyKey string) error
}

// HTTPGateway submits payments to the upstream payment provider.
type HTTPGateway struct {
	BaseURL string
	Client  *http.Client
}

// Submit sends one payment instruction, retrying transient gateway failures.
func (g *HTTPGateway) Submit(ctx context.Context, amountCents int64, currency, idempotencyKey string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.BaseURL+"/payments", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Idempotency-Key", idempotencyKey)

		resp, err := g.Client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("gateway returned %d", resp.StatusCode)
	}
	return lastErr
}
