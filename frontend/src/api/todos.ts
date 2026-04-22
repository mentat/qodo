import { auth } from '../firebase';
import type { Todo, TodoCreate, TodoUpdate } from '../types/todo';
import { API_BASE } from './base';

const BASE = `${API_BASE}/api/todos`;

async function headers(): Promise<HeadersInit> {
  const token = await auth.currentUser?.getIdToken();
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
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
