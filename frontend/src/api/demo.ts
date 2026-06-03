import { API_BASE, authHeaders, handleResponse } from './base';

const BASE = `${API_BASE}/api/demo`;

export async function seedDemo(): Promise<{ seeded: boolean }> {
  const res = await fetch(`${BASE}/seed`, { method: 'POST', headers: await authHeaders() });
  return handleResponse<{ seeded: boolean }>(res);
}

export async function resetDemo(): Promise<{ status: string }> {
  const res = await fetch(`${BASE}/reset`, { method: 'POST', headers: await authHeaders() });
  return handleResponse<{ status: string }>(res);
}
