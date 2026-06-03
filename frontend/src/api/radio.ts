import { API_BASE, authHeaders, handleResponse } from './base';
import type { Track } from '../types/radio';

export async function fetchTracks(): Promise<Track[]> {
  const res = await fetch(`${API_BASE}/api/radio/tracks`, { headers: await authHeaders() });
  return handleResponse<Track[]>(res);
}
