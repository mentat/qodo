// Shared types mirroring api/services/invoices.go. The Firestore docs and
// REST payloads use these shapes.
//
// All monetary values are whole cents — never floats. Use formatCents() to
// render them.

export type InvoiceStatus =
  | 'draft'
  | 'pending_approval'
  | 'approved'
  | 'scheduled'
  | 'paid'
  | 'rejected';

export type ApprovalDecisionState = 'pending' | 'approved' | 'rejected';

export interface InvoiceLine {
  description: string;
  glCode: string;
  amountCents: number;
}

export interface ApprovalStep {
  role: string;
  approverId: string;
  decision: ApprovalDecisionState;
  note?: string;
  auto: boolean;
  decidedAt: string | null;
}

export interface Invoice {
  id: string;
  vendorName: string;
  invoiceNumber: string;
  entityId: string;
  currencyCode: string;
  amountCents: number;
  lines: InvoiceLine[];
  status: InvoiceStatus;
  approvalChain: ApprovalStep[];
  dueDate: string | null;
  erpDocumentId?: string;
  paymentRef?: string;
  userId: string;
  createdAt: string;
  updatedAt: string;
}

export type InvoiceCreate = Pick<Invoice, 'vendorName' | 'entityId' | 'amountCents'> &
  Partial<Pick<Invoice, 'invoiceNumber' | 'currencyCode' | 'lines' | 'dueDate'>>;

export interface ApprovalTier {
  thresholdCents: number;
  role: string;
}

export interface ApprovalPolicy {
  delegatedLimitCents: number;
  tiers: ApprovalTier[];
}

// formatCents renders whole cents as a currency string.
export function formatCents(cents: number, currencyCode = 'USD'): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currencyCode,
  }).format(cents / 100);
}
