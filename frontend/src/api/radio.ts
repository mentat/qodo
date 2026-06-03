import { API_BASE, authHeaders, handleResponse } from './base';
import type { Track } from '../types/radio';

export async function fetchTracks(): Promise<Track[]> {
  const res = await fetch(`${API_BASE}/api/radio/tracks`, { headers: await authHeaders() });
  const tracks = await handleResponse<Track[]>(res);
  // The API returns a relative proxy path (/api/radio/stream?id=…); resolve it
  // against API_BASE so the <audio> element streams from our own origin.
  return tracks.map((t) => ({ ...t, url: t.url.startsWith('http') ? t.url : `${API_BASE}${t.url}` }));
}
