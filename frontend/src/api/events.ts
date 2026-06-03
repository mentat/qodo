import { API_BASE, authHeaders, handleResponse, expectOk } from './base';

const BASE = `${API_BASE}/api/events`;

// The API speaks RFC3339 strings; the store converts Date <-> string.
export interface EventInput {
  title: string;
  description?: string;
  location?: string;
  start: string;
  end?: string;
  allDay?: boolean;
  color?: string;
}

// Raw event shape returned by the REST API (times as strings).
export interface EventDTO {
  id: string;
  userId: string;
  title: string;
  description: string;
  location: string;
  start: string;
  end: string;
  allDay: boolean;
  color: string;
  characterId?: string;
}

export async function createEvent(input: EventInput): Promise<EventDTO> {
  const res = await fetch(BASE, { method: 'POST', headers: await authHeaders(), body: JSON.stringify(input) });
  return handleResponse<EventDTO>(res);
}

export async function updateEvent(id: string, input: EventInput): Promise<EventDTO> {
  const res = await fetch(`${BASE}/${id}`, { method: 'PUT', headers: await authHeaders(), body: JSON.stringify(input) });
  return handleResponse<EventDTO>(res);
}

export async function moveEvent(id: string, start: string, end?: string): Promise<EventDTO> {
  const res = await fetch(`${BASE}/${id}/move`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ start, end }),
  });
  return handleResponse<EventDTO>(res);
}

export async function deleteEvent(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE', headers: await authHeaders() });
  await expectOk(res);
}
