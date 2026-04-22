import { create } from 'zustand';
import * as api from '../api/todos';
import type { Todo, TodoCreate, TodoUpdate } from '../types/todo';

interface TodoState {
  todos: Todo[];
  // searchHits holds the server-side stemmed-search result when
  // searchQuery is non-empty. The UI reads from this in place of todos
  // for the rendered list. When searchQuery is empty, searchHits is null
  // and the UI falls back to the full todos list.
  searchHits: Todo[] | null;
  searchLoading: boolean;
  loading: boolean;

  // Filters
  searchQuery: string;
  priorityFilter: string | null;
  categoryFilter: string | null;
  statusFilter: string;

  // Actions
  fetchTodos: () => Promise<void>;
  addTodo: (data: TodoCreate) => Promise<void>;
  updateTodo: (id: string, data: TodoUpdate) => Promise<void>;
  toggleTodo: (id: string, completed: boolean) => Promise<void>;
  removeTodo: (id: string) => Promise<void>;
  reorderTodos: (startIndex: number, endIndex: number) => Promise<void>;
  runSearch: () => Promise<void>;

  // Filter setters
  setSearchQuery: (query: string) => void;
  setPriorityFilter: (priority: string | null) => void;
  setCategoryFilter: (category: string | null) => void;
  setStatusFilter: (status: string) => void;
}

export const useTodoStore = create<TodoState>((set, get) => ({
  todos: [],
  searchHits: null,
  searchLoading: false,
  loading: false,
  searchQuery: '',
  priorityFilter: null,
  categoryFilter: null,
  statusFilter: 'all',

  fetchTodos: async () => {
    set({ loading: true });
    try {
      const todos = await api.fetchTodos();
      set({ todos });
    } finally {
      set({ loading: false });
    }
  },

  addTodo: async (data) => {
    const todo = await api.createTodo(data);
    set((s) => ({ todos: [...s.todos, todo] }));
  },

  updateTodo: async (id, data) => {
    const updated = await api.updateTodo(id, data);
    set((s) => ({ todos: s.todos.map((t) => (t.id === id ? updated : t)) }));
  },

  toggleTodo: async (id, completed) => {
    // Optimistic update
    set((s) => ({ todos: s.todos.map((t) => (t.id === id ? { ...t, completed } : t)) }));
    try {
      await api.patchTodo(id, { completed });
    } catch {
      // Revert on failure
      set((s) => ({ todos: s.todos.map((t) => (t.id === id ? { ...t, completed: !completed } : t)) }));
      throw new Error('Failed to toggle todo');
    }
  },

  removeTodo: async (id) => {
    const prev = get().todos;
    set((s) => ({ todos: s.todos.filter((t) => t.id !== id) }));
    try {
      await api.deleteTodo(id);
    } catch {
      set({ todos: prev });
      throw new Error('Failed to delete todo');
    }
  },

  reorderTodos: async (startIndex, endIndex) => {
    const { todos } = get();
    const reordered = [...todos];
    const [moved] = reordered.splice(startIndex, 1);
    reordered.splice(endIndex, 0, moved);

    const items = reordered.map((t, i) => ({ id: t.id, position: i }));
    const updatedTodos = reordered.map((t, i) => ({ ...t, position: i }));
    set({ todos: updatedTodos });

    try {
      await api.reorderTodos(items);
    } catch {
      set({ todos });
      throw new Error('Failed to reorder');
    }
  },

  runSearch: async () => {
    const { searchQuery, statusFilter, priorityFilter } = get();
    const q = searchQuery.trim();
    if (!q) {
      set({ searchHits: null, searchLoading: false });
      return;
    }
    set({ searchLoading: true });
    try {
      const opts: { completed?: boolean; priority?: string } = {};
      if (statusFilter === 'active') opts.completed = false;
      else if (statusFilter === 'done') opts.completed = true;
      if (priorityFilter) opts.priority = priorityFilter;
      const hits = await api.searchTodos(q, opts);
      // Drop results if the query changed while we were fetching.
      if (get().searchQuery.trim() !== q) return;
      set({ searchHits: hits });
    } catch (err) {
      console.error('search failed', err);
      set({ searchHits: [] });
    } finally {
      set({ searchLoading: false });
    }
  },

  setSearchQuery: (searchQuery) => {
    set({ searchQuery });
    // Fire-and-forget; the UI reflects the result via searchHits.
    void get().runSearch();
  },
  setPriorityFilter: (priorityFilter) => {
    set({ priorityFilter });
    void get().runSearch();
  },
  setCategoryFilter: (categoryFilter) => set({ categoryFilter }),
  setStatusFilter: (statusFilter) => {
    set({ statusFilter });
    void get().runSearch();
  },
}));

// Pure filter function — used with useMemo in components to avoid
// infinite re-render loops with React 19's useSyncExternalStore
export function filterTodos(
  todos: Todo[],
  searchQuery: string,
  priorityFilter: string | null,
  categoryFilter: string | null,
  statusFilter: string,
): Todo[] {
  let result = todos;
  if (searchQuery) {
    const q = searchQuery.toLowerCase();
    result = result.filter(
      (t) => t.title.toLowerCase().includes(q) || t.description.toLowerCase().includes(q)
    );
  }
  if (priorityFilter) {
    result = result.filter((t) => t.priority === priorityFilter);
  }
  if (categoryFilter) {
    result = result.filter((t) => t.category === categoryFilter);
  }
  if (statusFilter === 'active') {
    result = result.filter((t) => !t.completed);
  } else if (statusFilter === 'done') {
    result = result.filter((t) => t.completed);
  }
  return result;
}

export function deriveCategories(todos: Todo[]): string[] {
  return [...new Set(todos.map((t) => t.category).filter(Boolean))];
}
