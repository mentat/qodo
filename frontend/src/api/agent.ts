import { auth } from '../firebase';
import { API_BASE } from './base';

const BASE = `${API_BASE}/api/agent`;

async function headers(): Promise<HeadersInit> {
  const token = await auth.currentUser?.getIdToken();
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

export type ChatRole = 'user' | 'assistant' | 'tool' | 'system';

export interface ChatMessage {
  id: string;
  userId: string;
  role: ChatRole;
  content: string;
  screened?: boolean;
  createdAt: string;
}

export interface ToolCall {
  name: string;
  args?: Record<string, unknown>;
  result?: string;
}

export interface ChatResponse {
  reply: string;
  toolCalls?: ToolCall[];
  screened?: boolean;
  reason?: string;
  messages?: ChatMessage[];
}

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
  return res.json();
}

export async function sendMessage(message: string): Promise<ChatResponse> {
  const res = await fetch(`${BASE}/chat`, {
    method: 'POST',
    headers: await headers(),
    body: JSON.stringify({ message }),
  });
  return handle<ChatResponse>(res);
}

export async function fetchHistory(limit = 50): Promise<ChatMessage[]> {
  const res = await fetch(`${BASE}/history?limit=${limit}`, { headers: await headers() });
  const body = await handle<{ messages: ChatMessage[] | null }>(res);
  return body.messages ?? [];
}

export async function clearHistory(): Promise<void> {
  const res = await fetch(`${BASE}/history`, {
    method: 'DELETE',
    headers: await headers(),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
}
