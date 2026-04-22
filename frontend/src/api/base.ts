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
