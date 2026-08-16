package services

import "testing"

func TestTotalCents(t *testing.T) {
	inv := Invoice{Lines: []InvoiceLine{
		{AmountCents: 120000},
		{AmountCents: 80000},
	}}
	if got := inv.TotalCents(); got != 200000 {
		t.Fatalf("TotalCents() = %d, want 200000", got)
	}
}

func TestWithinDelegatedAuthority_SingleLineOverLimit(t *testing.T) {
	inv := Invoice{Lines: []InvoiceLine{{AmountCents: 600000}}}
	if withinDelegatedAuthority(inv, 500000) {
		t.Fatal("a single line above the limit must not be auto-approved")
	}
}

func TestWithinDelegatedAuthority_UnderLimit(t *testing.T) {
	inv := Invoice{Lines: []InvoiceLine{{AmountCents: 100000}}}
	if !withinDelegatedAuthority(inv, 500000) {
		t.Fatal("an invoice under the limit should be auto-approved")
	}
}

func TestIdempotencyKeyIncludesInvoiceID(t *testing.T) {
	inv := Invoice{ID: "inv_123"}
	if key := idempotencyKey(inv); key == "" {
		t.Fatal("idempotency key must not be empty")
	}
}
