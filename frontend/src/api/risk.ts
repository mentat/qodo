import { API_BASE, authHeaders, handleResponse } from './base';
import type { GameState, Stats, AttackResult, Difficulty } from '../types/risk';
import type { TerritoryID } from '../components/Risk/board';

export async function fetchGame(): Promise<GameState | null> {
  const res = await fetch(`${API_BASE}/api/risk/`, { headers: await authHeaders() });
  if (res.status === 404) return null;
  return handleResponse<GameState>(res);
}

export async function fetchStats(): Promise<Stats> {
  const res = await fetch(`${API_BASE}/api/risk/stats`, { headers: await authHeaders() });
  return handleResponse<Stats>(res);
}

export async function newGame(difficulty: Difficulty, playerCount: number): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/new`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ difficulty, playerCount }),
  });
  return handleResponse<GameState>(res);
}

export async function placeInitial(territory: TerritoryID): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/place-initial`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ territory }),
  });
  return handleResponse<GameState>(res);
}

export async function placeArmies(territory: TerritoryID, count: number): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/place`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ territory, count }),
  });
  return handleResponse<GameState>(res);
}

export async function tradeCards(cardIds: string[]): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/trade`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ cardIds }),
  });
  return handleResponse<GameState>(res);
}

export interface AttackResponse {
  state: GameState;
  rounds: AttackResult[];
}

export async function attack(
  from: TerritoryID,
  to: TerritoryID,
  mode: 'single' | 'blitz' = 'blitz',
): Promise<AttackResponse> {
  const res = await fetch(`${API_BASE}/api/risk/attack`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ from, to, mode }),
  });
  return handleResponse<AttackResponse>(res);
}

export async function resolvePostConquest(count: number): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/post-conquest`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ count }),
  });
  return handleResponse<GameState>(res);
}

export async function fortify(from: TerritoryID, to: TerritoryID, count: number): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/fortify`, {
    method: 'POST',
    headers: await authHeaders(),
    body: JSON.stringify({ from, to, count }),
  });
  return handleResponse<GameState>(res);
}

export async function endPhase(): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/end-phase`, {
    method: 'POST',
    headers: await authHeaders(),
  });
  return handleResponse<GameState>(res);
}

export async function skipFortify(): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/skip-fortify`, {
    method: 'POST',
    headers: await authHeaders(),
  });
  return handleResponse<GameState>(res);
}

export async function surrender(): Promise<GameState> {
  const res = await fetch(`${API_BASE}/api/risk/surrender`, {
    method: 'POST',
    headers: await authHeaders(),
  });
  return handleResponse<GameState>(res);
}
