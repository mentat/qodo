package services_test

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/mentat/qodo/api/services"
)

// testPolicy: self-approval to $1,000, controller above that, CFO above
// $25,000. Mirrors services.DefaultApprovalPolicy.
var testPolicy = services.ApprovalPolicy{
	DelegatedLimitCents: 100_000,
	Tiers: []services.ApprovalTier{
		{ThresholdCents: 100_000, Role: "controller"},
		{ThresholdCents: 2_500_000, Role: "cfo"},
	},
}

func invoiceOf(amountCents int64, lines ...int64) services.Invoice {
	inv := services.Invoice{
		VendorName:   "Acme Supply Co",
		EntityID:     "entity-us-1",
		CurrencyCode: "USD",
		AmountCents:  amountCents,
	}
	if len(lines) == 0 {
		lines = []int64{amountCents}
	}
	for _, amt := range lines {
		inv.Lines = append(inv.Lines, services.InvoiceLine{
			Description: "line",
			GLCode:      "6000",
			AmountCents: amt,
		})
	}
	return inv
}

func TestEvaluateApproval_WithinDelegatedAuthority(t *testing.T) {
	// $250 — comfortably inside the $1,000 delegated limit.
	got := services.EvaluateApproval(invoiceOf(25_000), testPolicy)
	if !got.AutoApproved {
		t.Fatalf("expected auto-approval for $250 invoice, got %+v", got)
	}
	if len(got.RequiredRoles) != 0 {
		t.Fatalf("auto-approved invoice should require no roles, got %v", got.RequiredRoles)
	}
}

func TestEvaluateApproval_EscalatesToController(t *testing.T) {
	// $4,000 — past the delegated limit, below the CFO tier.
	got := services.EvaluateApproval(invoiceOf(400_000), testPolicy)
	if got.AutoApproved {
		t.Fatal("expected $4,000 invoice to require manual approval")
	}
	if len(got.RequiredRoles) != 1 || got.RequiredRoles[0] != "controller" {
		t.Fatalf("expected [controller], got %v", got.RequiredRoles)
	}
}

func TestEvaluateApproval_EscalatesToCFO(t *testing.T) {
	// $40,000 — past both tiers, so both must sign off in order.
	got := services.EvaluateApproval(invoiceOf(4_000_000), testPolicy)
	if got.AutoApproved {
		t.Fatal("expected $40,000 invoice to require manual approval")
	}
	want := []string{"controller", "cfo"}
	if len(got.RequiredRoles) != len(want) {
		t.Fatalf("expected %v, got %v", want, got.RequiredRoles)
	}
	for i := range want {
		if got.RequiredRoles[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got.RequiredRoles)
		}
	}
}

func TestEvaluateApproval_ZeroLimitNeverAutoApproves(t *testing.T) {
	strict := services.ApprovalPolicy{Tiers: testPolicy.Tiers}
	got := services.EvaluateApproval(invoiceOf(5_000), strict)
	if got.AutoApproved {
		t.Fatal("a zero delegated limit must never auto-approve")
	}
}

func TestAllocateCents_SumsToTotal(t *testing.T) {
	cases := []struct {
		name    string
		total   int64
		weights []int64
	}{
		{"even split", 30_000, []int64{1, 1, 1}},
		{"indivisible thirds", 10_000, []int64{1, 1, 1}},
		{"weighted", 99_999, []int64{5, 3, 2}},
		{"single bucket", 12_345, []int64{7}},
		{"one cent, many buckets", 1, []int64{1, 1, 1, 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := services.AllocateCents(tc.total, tc.weights)
			if len(got) != len(tc.weights) {
				t.Fatalf("expected %d buckets, got %d", len(tc.weights), len(got))
			}
			var sum int64
			for _, v := range got {
				sum += v
			}
			if sum != tc.total {
				t.Fatalf("allocation %v sums to %d, want %d", got, sum, tc.total)
			}
		})
	}
}

func TestAllocateCents_Degenerate(t *testing.T) {
	if got := services.AllocateCents(500, nil); len(got) != 0 {
		t.Fatalf("no weights should allocate nothing, got %v", got)
	}
	got := services.AllocateCents(500, []int64{0, 0})
	for _, v := range got {
		if v != 0 {
			t.Fatalf("zero weights should allocate nothing, got %v", got)
		}
	}
}

// The tests below talk to real Firestore, gated on the same credentials the
// API uses locally — same convention as todos_test.go. Writes go to the
// `invoices_test` collection under a unique userID per test, so nothing the
// app reads is ever touched.

func newTestInvoiceService(t *testing.T) (*services.InvoiceService, *services.StubPaymentGateway, func()) {
	t.Helper()
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		t.Skip("GOOGLE_APPLICATION_CREDENTIALS not set")
	}
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = "qodo-demo"
	}
	fs, err := firestore.NewClient(context.Background(), project)
	if err != nil {
		t.Fatalf("firestore client: %v", err)
	}
	gw := &services.StubPaymentGateway{}
	svc := services.NewInvoiceService(fs).
		WithCollection("invoices_test").
		WithPolicy(testPolicy).
		WithIntegrations(services.StubERPClient{}, gw)
	return svc, gw, func() { fs.Close() }
}

func TestInvoiceService_ApproveThenPay(t *testing.T) {
	svc, gw, closeFn := newTestInvoiceService(t)
	defer closeFn()
	ctx := context.Background()
	uid := uniqueUID(t)

	inv, err := svc.Create(ctx, uid, services.InvoiceCreateInput{
		VendorName:    "Acme Supply Co",
		InvoiceNumber: "INV-4471",
		EntityID:      "entity-us-1",
		AmountCents:   400_000,
		Lines: []services.InvoiceLine{
			{Description: "Consulting", GLCode: "6000", AmountCents: 400_000},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if inv.Status != services.InvoiceStatusPendingApproval {
		t.Fatalf("expected pending_approval, got %s", inv.Status)
	}

	approved, err := svc.Decide(ctx, uid, inv.ID, true, "checked against PO")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if approved.Status != services.InvoiceStatusApproved {
		t.Fatalf("expected approved, got %s", approved.Status)
	}

	paid, err := svc.SubmitPayment(ctx, uid, inv.ID)
	if err != nil {
		t.Fatalf("submit payment: %v", err)
	}
	if paid.Status != services.InvoiceStatusScheduled {
		t.Fatalf("expected scheduled, got %s", paid.Status)
	}
	if paid.PaymentRef == "" || paid.ERPDocumentID == "" {
		t.Fatalf("expected payment ref and ERP doc id, got %+v", paid)
	}
	if n := gw.SettledCount(); n != 1 {
		t.Fatalf("expected exactly 1 settled payment, got %d", n)
	}
}

func TestInvoiceService_RejectsUnbalancedLines(t *testing.T) {
	svc, _, closeFn := newTestInvoiceService(t)
	defer closeFn()

	_, err := svc.Create(context.Background(), uniqueUID(t), services.InvoiceCreateInput{
		VendorName:  "Acme Supply Co",
		EntityID:    "entity-us-1",
		AmountCents: 400_000,
		Lines: []services.InvoiceLine{
			{Description: "Consulting", GLCode: "6000", AmountCents: 350_000},
		},
	})
	if err == nil {
		t.Fatal("expected lines that don't sum to the total to be rejected")
	}
}
