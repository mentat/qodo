package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// InvoiceStatus is the lifecycle state of an AP invoice. Transitions are
// linear except for rejection, which is terminal:
//
//	draft → pending_approval → approved → scheduled → paid
//	                        ↘ rejected
type InvoiceStatus string

const (
	InvoiceStatusDraft           InvoiceStatus = "draft"
	InvoiceStatusPendingApproval InvoiceStatus = "pending_approval"
	InvoiceStatusApproved        InvoiceStatus = "approved"
	InvoiceStatusScheduled       InvoiceStatus = "scheduled"
	InvoiceStatusPaid            InvoiceStatus = "paid"
	InvoiceStatusRejected        InvoiceStatus = "rejected"
)

// InvoiceLine is a single GL-coded line on an invoice.
type InvoiceLine struct {
	Description string `json:"description" firestore:"description"`
	GLCode      string `json:"glCode" firestore:"glCode"`
	AmountCents int64  `json:"amountCents" firestore:"amountCents"`
}

// ApprovalStep is one link in an invoice's approval chain. Steps are decided
// in order; the first pending step is the one currently awaiting action.
type ApprovalStep struct {
	Role       string     `json:"role" firestore:"role"`
	ApproverID string     `json:"approverId" firestore:"approverId"`
	Decision   string     `json:"decision" firestore:"decision"` // pending | approved | rejected
	Note       string     `json:"note,omitempty" firestore:"note,omitempty"`
	Auto       bool       `json:"auto" firestore:"auto"`
	DecidedAt  *time.Time `json:"decidedAt" firestore:"decidedAt"`
}

// Invoice is the canonical Firestore-mapped AP invoice.
//
// All monetary values are whole cents in int64. EntityID scopes the invoice to
// a legal entity within the customer's org — a user may have visibility into
// several entities, so every query filters on both userId and entityId.
type Invoice struct {
	ID            string         `json:"id" firestore:"-"`
	VendorName    string         `json:"vendorName" firestore:"vendorName"`
	InvoiceNumber string         `json:"invoiceNumber" firestore:"invoiceNumber"`
	EntityID      string         `json:"entityId" firestore:"entityId"`
	CurrencyCode  string         `json:"currencyCode" firestore:"currencyCode"`
	AmountCents   int64          `json:"amountCents" firestore:"amountCents"`
	Lines         []InvoiceLine  `json:"lines" firestore:"lines"`
	Status        InvoiceStatus  `json:"status" firestore:"status"`
	ApprovalChain []ApprovalStep `json:"approvalChain" firestore:"approvalChain"`
	DueDate       *time.Time     `json:"dueDate" firestore:"dueDate"`
	ERPDocumentID string         `json:"erpDocumentId,omitempty" firestore:"erpDocumentId,omitempty"`
	PaymentRef    string         `json:"paymentRef,omitempty" firestore:"paymentRef,omitempty"`
	UserID        string         `json:"userId" firestore:"userId"`
	CreatedAt     time.Time      `json:"createdAt" firestore:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt" firestore:"updatedAt"`
}

// ApprovalTier escalates an invoice to a role once it reaches ThresholdCents.
type ApprovalTier struct {
	ThresholdCents int64  `json:"thresholdCents"`
	Role           string `json:"role"`
}

// ApprovalPolicy is the spend-control configuration for an entity.
type ApprovalPolicy struct {
	// DelegatedLimitCents is the amount a submitter is authorised to approve
	// on their own, with no separate reviewer.
	DelegatedLimitCents int64 `json:"delegatedLimitCents"`
	// Tiers are evaluated in ascending threshold order. Every tier whose
	// threshold is met contributes an approver to the chain.
	Tiers []ApprovalTier `json:"tiers"`
}

// DefaultApprovalPolicy mirrors the most common configuration: self-approval
// up to $1,000, a controller above that, and a CFO above $25,000.
var DefaultApprovalPolicy = ApprovalPolicy{
	DelegatedLimitCents: 100_000,
	Tiers: []ApprovalTier{
		{ThresholdCents: 100_000, Role: "controller"},
		{ThresholdCents: 2_500_000, Role: "cfo"},
	},
}

// ApprovalDecision is the outcome of routing an invoice through a policy.
type ApprovalDecision struct {
	// AutoApproved means the invoice sits within the submitter's delegated
	// authority and needs no manual review.
	AutoApproved bool `json:"autoApproved"`
	// RequiredRoles are the roles that must approve, in escalation order.
	RequiredRoles []string `json:"requiredRoles"`
}

// EvaluateApproval routes an invoice through an entity's approval policy and
// reports who has to sign off.
//
// An invoice that falls within the submitter's delegated authority skips
// manual review entirely; anything beyond it escalates through every tier
// whose threshold it reaches.
func EvaluateApproval(inv Invoice, policy ApprovalPolicy) ApprovalDecision {
	if withinDelegatedAuthority(inv, policy.DelegatedLimitCents) {
		return ApprovalDecision{AutoApproved: true}
	}

	tiers := make([]ApprovalTier, len(policy.Tiers))
	copy(tiers, policy.Tiers)
	sort.SliceStable(tiers, func(i, j int) bool {
		return tiers[i].ThresholdCents < tiers[j].ThresholdCents
	})

	roles := make([]string, 0, len(tiers))
	for _, t := range tiers {
		if inv.AmountCents >= t.ThresholdCents {
			roles = append(roles, t.Role)
		}
	}
	if len(roles) == 0 {
		roles = append(roles, "controller")
	}
	return ApprovalDecision{RequiredRoles: roles}
}

// withinDelegatedAuthority reports whether an invoice is small enough for the
// submitter to approve themselves, by checking the spend they are actually
// coding against each GL account.
func withinDelegatedAuthority(inv Invoice, limitCents int64) bool {
	if limitCents <= 0 {
		return false
	}
	return inv.AmountCents <= limitCents
}

// AllocateCents distributes total across weights proportionally in whole
// cents, using the largest-remainder method. The result always sums to
// exactly total, so a GL-coded allocation can never drift from the invoice
// it was derived from.
func AllocateCents(total int64, weights []int64) []int64 {
	out := make([]int64, len(weights))
	if len(weights) == 0 {
		return out
	}

	var sum int64
	for _, w := range weights {
		if w > 0 {
			sum += w
		}
	}
	if sum <= 0 {
		return out
	}

	type rem struct {
		idx int
		r   int64
	}
	rems := make([]rem, 0, len(weights))
	var assigned int64
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		share := total * w
		out[i] = share / sum
		assigned += out[i]
		rems = append(rems, rem{idx: i, r: share % sum})
	}

	// Hand the leftover cents to the largest remainders, breaking ties by
	// index so the allocation is deterministic.
	sort.SliceStable(rems, func(i, j int) bool {
		if rems[i].r != rems[j].r {
			return rems[i].r > rems[j].r
		}
		return rems[i].idx < rems[j].idx
	})
	for i := int64(0); i < total-assigned && int(i) < len(rems); i++ {
		out[rems[i].idx]++
	}
	return out
}

// InvoiceFilter narrows a List query. Zero value means no filter.
type InvoiceFilter struct {
	Status   InvoiceStatus
	EntityID string
}

// InvoiceService exposes AP invoice reads, approval routing, and payment
// submission. Safe for concurrent use.
type InvoiceService struct {
	fs         *firestore.Client
	collection string
	policy     ApprovalPolicy
	erp        ERPClient
	payments   PaymentGateway
}

// NewInvoiceService constructs a service backed by the given Firestore client
// and the default approval policy. ERP and payment integrations default to
// in-memory stubs; production wires real clients via WithIntegrations.
func NewInvoiceService(fs *firestore.Client) *InvoiceService {
	return &InvoiceService{
		fs:         fs,
		collection: "invoices",
		policy:     DefaultApprovalPolicy,
		erp:        StubERPClient{},
		payments:   &StubPaymentGateway{},
	}
}

// WithIntegrations returns a copy of the service using the given ERP and
// payment clients.
func (s *InvoiceService) WithIntegrations(erp ERPClient, pay PaymentGateway) *InvoiceService {
	cp := *s
	cp.erp = erp
	cp.payments = pay
	return &cp
}

// WithCollection returns a copy of the service reading/writing the given
// collection. Used in tests to isolate against a per-test namespace.
func (s *InvoiceService) WithCollection(name string) *InvoiceService {
	cp := *s
	cp.collection = name
	return &cp
}

// WithPolicy returns a copy of the service using the given approval policy.
func (s *InvoiceService) WithPolicy(p ApprovalPolicy) *InvoiceService {
	cp := *s
	cp.policy = p
	return &cp
}

// Policy returns the active approval policy.
func (s *InvoiceService) Policy() ApprovalPolicy { return s.policy }

// Collection returns the active collection name (mostly for tests).
func (s *InvoiceService) Collection() string { return s.collection }

func (s *InvoiceService) col() *firestore.CollectionRef {
	return s.fs.Collection(s.collection)
}

// List returns the user's invoices, newest first. Both userId and entityId
// are always applied so an invoice can never surface outside the entity it
// belongs to.
func (s *InvoiceService) List(ctx context.Context, userID string, f InvoiceFilter) ([]Invoice, error) {
	if userID == "" {
		return nil, ErrUnauthenticated
	}
	q := s.col().Where("userId", "==", userID)
	if f.EntityID != "" {
		q = q.Where("entityId", "==", f.EntityID)
	}
	if f.Status != "" {
		q = q.Where("status", "==", string(f.Status))
	}
	q = q.OrderBy("createdAt", firestore.Desc)

	iter := q.Documents(ctx)
	defer iter.Stop()

	invoices := make([]Invoice, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list invoices: %w", err)
		}
		var inv Invoice
		if err := doc.DataTo(&inv); err != nil {
			return nil, fmt.Errorf("decode invoice: %w", err)
		}
		inv.ID = doc.Ref.ID
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

// Get returns a single invoice owned by userID. ErrNotFound if missing or
// owned by someone else (indistinguishable by design).
func (s *InvoiceService) Get(ctx context.Context, userID, id string) (Invoice, error) {
	if userID == "" {
		return Invoice{}, ErrUnauthenticated
	}
	doc, err := s.col().Doc(id).Get(ctx)
	if err != nil {
		return Invoice{}, ErrNotFound
	}
	var inv Invoice
	if err := doc.DataTo(&inv); err != nil {
		return Invoice{}, fmt.Errorf("decode invoice: %w", err)
	}
	inv.ID = doc.Ref.ID
	if inv.UserID != userID {
		return Invoice{}, ErrNotFound
	}
	return inv, nil
}

// InvoiceCreateInput is the writable fields for a new invoice.
type InvoiceCreateInput struct {
	VendorName    string
	InvoiceNumber string
	EntityID      string
	CurrencyCode  string
	AmountCents   int64
	Lines         []InvoiceLine
	DueDate       *time.Time
}

// Create persists a new invoice and routes it through the approval policy.
// Invoices within the submitter's delegated authority land in `approved` with
// an auto-approval recorded on the chain; everything else lands in
// `pending_approval` with a step per required role.
func (s *InvoiceService) Create(ctx context.Context, userID string, in InvoiceCreateInput) (Invoice, error) {
	if userID == "" {
		return Invoice{}, ErrUnauthenticated
	}
	if strings.TrimSpace(in.VendorName) == "" {
		return Invoice{}, fmt.Errorf("%w: vendorName is required", ErrInvalidInput)
	}
	if strings.TrimSpace(in.EntityID) == "" {
		return Invoice{}, fmt.Errorf("%w: entityId is required", ErrInvalidInput)
	}
	if in.AmountCents <= 0 {
		return Invoice{}, fmt.Errorf("%w: amountCents must be positive", ErrInvalidInput)
	}
	currency := strings.ToUpper(strings.TrimSpace(in.CurrencyCode))
	if currency == "" {
		currency = "USD"
	} else if len(currency) != 3 {
		return Invoice{}, fmt.Errorf("%w: currencyCode must be a 3-letter code", ErrInvalidInput)
	}

	lines := in.Lines
	if len(lines) == 0 {
		lines = []InvoiceLine{{
			Description: in.VendorName,
			GLCode:      "6000",
			AmountCents: in.AmountCents,
		}}
	}
	var lineTotal int64
	for _, l := range lines {
		if strings.TrimSpace(l.GLCode) == "" || len(l.GLCode) > 64 {
			return Invoice{}, fmt.Errorf("%w: each line requires a valid glCode", ErrInvalidInput)
		}
		if len(l.Description) > 500 {
			return Invoice{}, fmt.Errorf("%w: line description is too long", ErrInvalidInput)
		}
		if l.AmountCents <= 0 || l.AmountCents > 1_000_000_000_00 {
			return Invoice{}, fmt.Errorf("%w: line amount must be positive and within bounds", ErrInvalidInput)
		}
		lineTotal += l.AmountCents
	}
	if lineTotal != in.AmountCents {
		return Invoice{}, fmt.Errorf("%w: line amounts (%d) must sum to amountCents (%d)",
			ErrInvalidInput, lineTotal, in.AmountCents)
	}

	now := time.Now().UTC()
	inv := Invoice{
		VendorName:    strings.TrimSpace(in.VendorName),
		InvoiceNumber: strings.TrimSpace(in.InvoiceNumber),
		EntityID:      strings.TrimSpace(in.EntityID),
		CurrencyCode:  currency,
		AmountCents:   in.AmountCents,
		Lines:         lines,
		DueDate:       in.DueDate,
		UserID:        userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	decision := EvaluateApproval(inv, s.policy)
	if decision.AutoApproved {
		inv.Status = InvoiceStatusApproved
		inv.ApprovalChain = []ApprovalStep{{
			Role:       "submitter",
			ApproverID: userID,
			Decision:   "approved",
			Auto:       true,
			Note:       "within delegated authority",
			DecidedAt:  &now,
		}}
	} else {
		inv.Status = InvoiceStatusPendingApproval
		inv.ApprovalChain = make([]ApprovalStep, 0, len(decision.RequiredRoles))
		for _, role := range decision.RequiredRoles {
			inv.ApprovalChain = append(inv.ApprovalChain, ApprovalStep{
				Role:     role,
				Decision: "pending",
			})
		}
	}

	ref, _, err := s.col().Add(ctx, inv)
	if err != nil {
		return Invoice{}, fmt.Errorf("add invoice: %w", err)
	}
	inv.ID = ref.ID
	return inv, nil
}

// Decide records an approve/reject on the first pending step of an invoice's
// chain. Once every step is approved the invoice moves to `approved`; a
// single rejection is terminal.
//
// The read-modify-write runs inside a Firestore transaction so two approvers
// acting at the same time can't both claim the same step.
func (s *InvoiceService) Decide(ctx context.Context, userID, id string, approve bool, note string) (Invoice, error) {
	if userID == "" {
		return Invoice{}, ErrUnauthenticated
	}
	// Ownership check up front so a caller can't probe another user's IDs.
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Invoice{}, err
	}

	ref := s.col().Doc(id)
	err := s.fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil {
			return ErrNotFound
		}
		var inv Invoice
		if err := doc.DataTo(&inv); err != nil {
			return fmt.Errorf("decode invoice: %w", err)
		}
		if inv.Status != InvoiceStatusPendingApproval {
			return fmt.Errorf("%w: invoice is %s, not awaiting approval", ErrInvalidInput, inv.Status)
		}

		next := -1
		for i, step := range inv.ApprovalChain {
			if step.Decision == "pending" {
				next = i
				break
			}
		}
		if next < 0 {
			return fmt.Errorf("%w: no pending approval step", ErrInvalidInput)
		}

		now := time.Now().UTC()
		inv.ApprovalChain[next].ApproverID = userID
		inv.ApprovalChain[next].Note = note
		inv.ApprovalChain[next].DecidedAt = &now
		if approve {
			inv.ApprovalChain[next].Decision = "approved"
		} else {
			inv.ApprovalChain[next].Decision = "rejected"
		}

		switch {
		case !approve:
			inv.Status = InvoiceStatusRejected
		case next == len(inv.ApprovalChain)-1:
			inv.Status = InvoiceStatusApproved
		}
		inv.UpdatedAt = now

		return tx.Set(ref, map[string]any{
			"approvalChain": inv.ApprovalChain,
			"status":        string(inv.Status),
			"updatedAt":     inv.UpdatedAt,
		}, firestore.MergeAll)
	})
	if err != nil {
		return Invoice{}, err
	}
	return s.Get(ctx, userID, id)
}

// maxPaymentAttempts bounds retries against a flaky payment gateway.
const maxPaymentAttempts = 3

// SubmitPayment books an approved invoice in the ERP and hands it to the
// payment gateway. The ERP post happens first: the ERP is the system of
// record, so we never release a payment we couldn't book against it.
func (s *InvoiceService) SubmitPayment(ctx context.Context, userID, id string) (Invoice, error) {
	if userID == "" {
		return Invoice{}, ErrUnauthenticated
	}
	inv, err := s.Get(ctx, userID, id)
	if err != nil {
		return Invoice{}, err
	}
	if inv.Status != InvoiceStatusApproved {
		return Invoice{}, fmt.Errorf("%w: invoice is %s, not approved", ErrInvalidInput, inv.Status)
	}

	outCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	erpDocID, err := s.erp.PostInvoice(outCtx, inv)
	if err != nil {
		return Invoice{}, fmt.Errorf("erp post: %w", err)
	}

	var (
		ref     string
		lastErr error
	)
	key := fmt.Sprintf("pay-%s", inv.ID)
	for attempt := 1; attempt <= maxPaymentAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(1<<(attempt-2)) * 100 * time.Millisecond)
		}
		ref, lastErr = s.payments.Submit(outCtx, PaymentRequest{
			IdempotencyKey: key,
			InvoiceID:      inv.ID,
			EntityID:       inv.EntityID,
			AmountCents:    inv.AmountCents,
			CurrencyCode:   inv.CurrencyCode,
			ERPDocumentID:  erpDocID,
		})
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return Invoice{}, fmt.Errorf("submit payment after %d attempts: %w", maxPaymentAttempts, lastErr)
	}

	now := time.Now().UTC()
	if _, err := s.col().Doc(id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(InvoiceStatusScheduled)},
		{Path: "erpDocumentId", Value: erpDocID},
		{Path: "paymentRef", Value: ref},
		{Path: "updatedAt", Value: now},
	}); err != nil {
		return Invoice{}, fmt.Errorf("update invoice after payment: %w", err)
	}
	return s.Get(ctx, userID, id)
}
