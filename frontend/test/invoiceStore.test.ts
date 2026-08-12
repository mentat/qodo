import { describe, it, expect } from 'bun:test';
import { awaitingApprovalTotal } from '../src/store/invoiceStore';
import { formatCents } from '../src/types/invoice';
import type { Invoice } from '../src/types/invoice';

function invoice(over: Partial<Invoice>): Invoice {
  return {
    id: 'x',
    vendorName: 'Acme Supply Co',
    invoiceNumber: 'INV-1',
    entityId: 'entity-us-1',
    currencyCode: 'USD',
    amountCents: 25_000,
    lines: [{ description: 'line', glCode: '6000', amountCents: 25_000 }],
    status: 'pending_approval',
    approvalChain: [],
    dueDate: null,
    userId: 'u',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...over,
  };
}

describe('awaitingApprovalTotal', () => {
  it('sums only invoices still awaiting approval', () => {
    const invoices: Invoice[] = [
      invoice({ id: '1', amountCents: 100_000, status: 'pending_approval' }),
      invoice({ id: '2', amountCents: 250_000, status: 'pending_approval' }),
      invoice({ id: '3', amountCents: 900_000, status: 'approved' }),
      invoice({ id: '4', amountCents: 500_000, status: 'paid' }),
      invoice({ id: '5', amountCents: 700_000, status: 'rejected' }),
    ];
    expect(awaitingApprovalTotal(invoices)).toBe(350_000);
  });

  it('is zero for an empty queue', () => {
    expect(awaitingApprovalTotal([])).toBe(0);
    expect(awaitingApprovalTotal([invoice({ status: 'paid' })])).toBe(0);
  });
});

describe('formatCents', () => {
  it('renders whole cents as currency without float drift', () => {
    expect(formatCents(0)).toBe('$0.00');
    expect(formatCents(1)).toBe('$0.01');
    expect(formatCents(100_000)).toBe('$1,000.00');
    expect(formatCents(2_500_099)).toBe('$25,000.99');
  });

  it('honours the invoice currency', () => {
    expect(formatCents(100_000, 'EUR')).toBe('€1,000.00');
  });
});
