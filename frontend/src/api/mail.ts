import { API_BASE, authHeaders, handleResponse, expectOk } from './base';
import type { Email } from '../types/mail';

const BASE = `${API_BASE}/api/emails`;

export interface SendInput {
  to: string;
  toName?: string;
  subject: string;
  body: string;
  threadId?: string;
  characterId?: string;
}

export async function sendEmail(input: SendInput): Promise<Email> {
  const res = await fetch(BASE, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify(input),
  });
  return handleResponse<Email>(res);
}

export async function markEmailRead(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}/read`, { method: 'POST', headers: await authHeaders() });
  await expectOk(res);
}

export async function setStarred(id: string, starred: boolean): Promise<void> {
  const res = await fetch(`${BASE}/${id}/star`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ starred }),
  });
  await expectOk(res);
}

export async function deleteEmail(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE', headers: await authHeaders() });
  await expectOk(res);
}
