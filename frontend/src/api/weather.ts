import { API_BASE, authHeaders, handleResponse } from './base';
import type { Forecast } from '../types/weather';

const BASE = `${API_BASE}/api/weather`;

export async function fetchForecast(location: string, days = 5): Promise<Forecast> {
  const params = new URLSearchParams({ location, days: String(days) });
  const res = await fetch(`${BASE}?${params.toString()}`, { headers: await authHeaders() });
  return handleResponse<Forecast>(res);
}
