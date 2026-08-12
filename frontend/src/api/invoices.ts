import { API_BASE, authHeaders, handleResponse } from './base';
import type { ApprovalPolicy, Invoice, InvoiceCreate, InvoiceStatus } from '../types/invoice';

const BASE = `${API_BASE}/api/invoices`;

export interface InvoiceQuery {
  status?: InvoiceStatus;
  entityId?: string;
}

export async function fetchInvoices(q: InvoiceQuery = {}): Promise<Invoice[]> {
  const params = new URLSearchParams();
  if (q.status) params.set('status', q.status);
  if (q.entityId) params.set('entityId', q.entityId);
  const suffix = params.toString() ? `?${params}` : '';
  const res = await fetch(`${BASE}${suffix}`, { headers: await authHeaders() });
  return handleResponse<Invoice[]>(res);
}

export async function fetchInvoice(id: string): Promise<Invoice> {
  const res = await fetch(`${BASE}/${id}`, { headers: await authHeaders() });
  return handleResponse<Invoice>(res);
}

export async function fetchApprovalPolicy(): Promise<ApprovalPolicy> {
  const res = await fetch(`${BASE}/policy`, { headers: await authHeaders() });
  return handleResponse<ApprovalPolicy>(res);
}

export async function createInvoice(data: InvoiceCreate): Promise<Invoice> {
  const res = await fetch(BASE, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse<Invoice>(res);
}

export async function approveInvoice(id: string, note = ''): Promise<Invoice> {
  const res = await fetch(`${BASE}/${id}/approve`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ note }),
  });
  return handleResponse<Invoice>(res);
}

export async function rejectInvoice(id: string, note = ''): Promise<Invoice> {
  const res = await fetch(`${BASE}/${id}/reject`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ note }),
  });
  return handleResponse<Invoice>(res);
}

export async function payInvoice(id: string): Promise<Invoice> {
  const res = await fetch(`${BASE}/${id}/pay`, {
    method: 'POST',
    headers: await authHeaders(),
  });
  return handleResponse<Invoice>(res);
}
