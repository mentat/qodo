// Single source of truth for the backend base URL.
//
// Defaults to http://localhost:4090 in dev (matches supervisord) and can be
// overridden at build time with VITE_API_URL — e.g. when Firebase Hosting
// serves the frontend and Cloud Run serves the API.
export const API_BASE: string =
  (import.meta.env.VITE_API_URL as string | undefined)?.replace(/\/$/, '') ||
  (typeof window !== 'undefined' && window.location.hostname === 'localhost'
    ? 'http://localhost:4090'
    : window.location.origin);

import { auth } from '../firebase';

// Shared request helpers used by the suite API clients (todos.ts predates
// these and keeps its own copies). The Firebase ID token is attached on
// every call so the Go middleware can resolve the user.
export async function authHeaders(): Promise<HeadersInit> {
  const token = await auth.currentUser?.getIdToken();
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

export async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error((body as { error?: string }).error || res.statusText);
  }
  return res.json() as Promise<T>;
}

export async function expectOk(res: Response): Promise<void> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error((body as { error?: string }).error || res.statusText);
  }
}
