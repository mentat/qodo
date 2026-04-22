import { create } from 'zustand';
import * as api from '../api/todos';
import type { Todo, TodoCreate, TodoUpdate } from '../types/todo';

interface TodoState {
  todos: Todo[];
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

  // Filter setters
  setSearchQuery: (query: string) => void;
  setPriorityFilter: (priority: string | null) => void;
  setCategoryFilter: (category: string | null) => void;
  setStatusFilter: (status: string) => void;
}

export const useTodoStore = create<TodoState>((set, get) => ({
  todos: [],
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

  setSearchQuery: (searchQuery) => set({ searchQuery }),
  setPriorityFilter: (priorityFilter) => set({ priorityFilter }),
  setCategoryFilter: (categoryFilter) => set({ categoryFilter }),
  setStatusFilter: (statusFilter) => set({ statusFilter }),
}));

// Derived selectors
export const useFilteredTodos = () =>
  useTodoStore((s) => {
    let result = s.todos;
    if (s.searchQuery) {
      const q = s.searchQuery.toLowerCase();
      result = result.filter(
        (t) => t.title.toLowerCase().includes(q) || t.description.toLowerCase().includes(q)
      );
    }
    if (s.priorityFilter) {
      result = result.filter((t) => t.priority === s.priorityFilter);
    }
    if (s.categoryFilter) {
      result = result.filter((t) => t.category === s.categoryFilter);
    }
    if (s.statusFilter === 'active') {
      result = result.filter((t) => !t.completed);
    } else if (s.statusFilter === 'done') {
      result = result.filter((t) => t.completed);
    }
    return result;
  });

export const useCategories = () =>
  useTodoStore((s) => [...new Set(s.todos.map((t) => t.category).filter(Boolean))]);
