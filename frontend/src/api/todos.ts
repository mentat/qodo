import { auth } from '../firebase';
import type { Todo, TodoCreate, TodoUpdate } from '../types/todo';
import { API_BASE } from './base';

const BASE = `${API_BASE}/api/todos`;

let cachedToken: string | null = null;

async function headers(): Promise<HeadersInit> {
  if (!cachedToken) {
    cachedToken = (await auth.currentUser?.getIdToken()) ?? null;
  }
  return {
    'Content-Type': 'application/json',
    ...(cachedToken ? { Authorization: `Bearer ${cachedToken}` } : {}),
  };
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
  return res.json();
}

export async function fetchTodos(): Promise<Todo[]> {
  const res = await fetch(BASE, { headers: await headers() });
  return handleResponse<Todo[]>(res);
}

export interface SearchOptions {
  completed?: boolean;
  priority?: string;
  limit?: number;
}

// searchTodos hits the server-side stemmed full-text index. Empty q returns
// all user todos (the server falls through to List) — the UI treats this
// endpoint as the single source of truth for the filtered list whenever a
// query is active.
export async function searchTodos(q: string, opts: SearchOptions = {}): Promise<Todo[]> {
  const params = new URLSearchParams({ q });
  if (opts.completed !== undefined) params.set('completed', String(opts.completed));
  if (opts.priority) params.set('priority', opts.priority);
  if (opts.limit) params.set('limit', String(opts.limit));
  const res = await fetch(`${BASE}/search?${params.toString()}`, { headers: await headers() });
  return handleResponse<Todo[]>(res);
}

export async function createTodo(data: TodoCreate): Promise<Todo> {
  const res = await fetch(BASE, {
    method: 'POST',
    headers: await headers(),
    body: JSON.stringify(data),
  });
  return handleResponse<Todo>(res);
}

export async function updateTodo(id: string, data: TodoUpdate): Promise<Todo> {
  const res = await fetch(`${BASE}/${id}`, {
    method: 'PUT',
    headers: await headers(),
    body: JSON.stringify(data),
  });
  return handleResponse<Todo>(res);
}

export async function patchTodo(id: string, data: Partial<TodoUpdate>): Promise<Todo> {
  const res = await fetch(`${BASE}/${id}`, {
    method: 'PATCH',
    headers: await headers(),
    body: JSON.stringify(data),
  });
  return handleResponse<Todo>(res);
}

export async function deleteTodo(id: string): Promise<void> {
  const res = await fetch(`${BASE}/${id}`, {
    method: 'DELETE',
    headers: await headers(),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
}

export async function reorderTodos(items: { id: string; position: number }[]): Promise<void> {
  const res = await fetch(`${BASE}/reorder`, {
    method: 'POST',
    headers: await headers(),
    body: JSON.stringify({ items }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
}
