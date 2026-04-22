import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Frontend and backend run as independent services. The frontend addresses
// the backend via VITE_API_URL (defaults to the dev-supervisord port 4090).
// The Go API already enables CORS for any origin, so no dev proxy is needed.
export default defineConfig({
  plugins: [react()],
});
