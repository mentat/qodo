import { create } from 'zustand';
import * as api from '../api/invoices';
import type { ApprovalPolicy, Invoice, InvoiceCreate, InvoiceStatus } from '../types/invoice';

interface InvoiceState {
  invoices: Invoice[];
  policy: ApprovalPolicy | null;
  entityId: string;
  status: InvoiceStatus | 'all';
  loading: boolean;
  /** IDs with an approve/reject/pay request in flight. */
  pending: string[];

  fetch: () => Promise<void>;
  fetchPolicy: () => Promise<void>;
  setEntity: (entityId: string) => void;
  setStatus: (status: InvoiceStatus | 'all') => void;
  add: (data: InvoiceCreate) => Promise<Invoice>;
  approve: (id: string, note?: string) => Promise<void>;
  reject: (id: string, note?: string) => Promise<void>;
  pay: (id: string) => Promise<void>;
}

export const useInvoiceStore = create<InvoiceState>((set, get) => ({
  invoices: [],
  policy: null,
  entityId: 'entity-us-1',
  status: 'all',
  loading: false,
  pending: [],

  fetch: async () => {
    const { entityId, status } = get();
    set({ loading: true });
    try {
      set({
        invoices: await api.fetchInvoices({
          entityId,
          status: status === 'all' ? undefined : status,
        }),
      });
    } finally {
      set({ loading: false });
    }
  },

  fetchPolicy: async () => {
    set({ policy: await api.fetchApprovalPolicy() });
  },

  setEntity: (entityId) => set({ entityId }),
  setStatus: (status) => set({ status }),

  add: async (data) => {
    const inv = await api.createInvoice(data);
    set((s) => ({ invoices: [inv, ...s.invoices] }));
    return inv;
  },

  approve: async (id, note = '') => {
    await runAction(set, get, id, () => api.approveInvoice(id, note));
  },

  reject: async (id, note = '') => {
    await runAction(set, get, id, () => api.rejectInvoice(id, note));
  },

  pay: async (id) => {
    await runAction(set, get, id, () => api.payInvoice(id));
  },
}));

// awaitingApprovalTotal sums the invoices still sitting in the approval queue.
export function awaitingApprovalTotal(invoices: Invoice[]): number {
  return invoices
    .filter((i) => i.status === 'pending_approval')
    .reduce((sum, i) => sum + i.amount, 0);
}

// runAction marks an invoice as in-flight, swaps in the server's post-state on
// success, and always clears the in-flight marker. Guards against a second
// request for the same invoice while one is outstanding.
async function runAction(
  set: (fn: (s: InvoiceState) => Partial<InvoiceState>) => void,
  get: () => InvoiceState,
  id: string,
  call: () => Promise<Invoice>,
): Promise<void> {
  if (get().pending.includes(id)) return;
  set((s) => ({ pending: [...s.pending, id] }));
  try {
    const updated = await call();
    set((s) => ({ invoices: s.invoices.map((i) => (i.id === id ? updated : i)) }));
  } finally {
    set((s) => ({ pending: s.pending.filter((p) => p !== id) }));
  }
}
