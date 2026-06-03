import { API_BASE, authHeaders, handleResponse, expectOk } from './base';
import type { Contact, ContactCreate } from '../types/contact';

const BASE = `${API_BASE}/api/contacts`;

export async function fetchContacts(): Promise<Contact[]> {
  const res = await fetch(BASE, { headers: await authHeaders() });
  return handleResponse<Contact[]>(res);
}

export async function searchContacts(q: string): Promise<Contact[]> {
  const res = await fetch(`${BASE}/search?q=${encodeURIComponent(q)}`, { headers: await authHeaders() });
  return handleResponse<Contact[]>(res);
}

export async function createContact(data: ContactCreate): Promise<Contact> {
  const res = await fetch(BASE, { method: 'POST', headers: await authHeaders(), body: JSON.stringify(data) });
  return handleResponse<Contact>(res);
}

export async function updateContact(id: string, data: Partial<ContactCreate>): Promise<Contact> {
  const res = await fetch(`${BASE}/${id}`, { method: 'PATCH', headers: await authHeaders(), body: JSON.stringify(data) });
  return handleResponse<Contact>(res);
}

export async function deleteContact(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE', headers: await authHeaders() });
  await expectOk(res);
}
