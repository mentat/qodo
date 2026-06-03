import { API_BASE, authHeaders, handleResponse, expectOk } from './base';
import type { Note, NoteCreate } from '../types/note';

const BASE = `${API_BASE}/api/notes`;

export async function fetchNotes(): Promise<Note[]> {
  const res = await fetch(BASE, { headers: await authHeaders() });
  return handleResponse<Note[]>(res);
}

export async function searchNotes(q: string): Promise<Note[]> {
  const res = await fetch(`${BASE}/search?q=${encodeURIComponent(q)}`, { headers: await authHeaders() });
  return handleResponse<Note[]>(res);
}

export async function createNote(data: NoteCreate): Promise<Note> {
  const res = await fetch(BASE, { method: 'POST', headers: await authHeaders(), body: JSON.stringify(data) });
  return handleResponse<Note>(res);
}

export async function updateNote(id: string, data: NoteCreate): Promise<Note> {
  const res = await fetch(`${BASE}/${id}`, { method: 'PUT', headers: await authHeaders(), body: JSON.stringify(data) });
  return handleResponse<Note>(res);
}

export async function deleteNote(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE', headers: await authHeaders() });
  await expectOk(res);
}
