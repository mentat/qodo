import { create } from 'zustand';
import { sendMessage, fetchHistory, clearHistory, type ChatMessage, type ToolCall } from '../api/agent';
import { useTodoStore } from './todoStore';

interface ChatState {
  messages: ChatMessage[];
  sending: boolean;
  loading: boolean;
  loaded: boolean;
  lastToolCalls: ToolCall[];
  error: string | null;

  load: () => Promise<void>;
  send: (text: string) => Promise<void>;
  reset: () => Promise<void>;
}

const TODO_MUTATING_TOOLS = new Set(['create_todo', 'update_todo', 'delete_todo']);

export const useChatStore = create<ChatState>((set, get) => ({
  messages: [],
  sending: false,
  loading: false,
  loaded: false,
  lastToolCalls: [],
  error: null,

  async load() {
    // Always re-sync on open so a page refresh (or a message sent in another
    // tab) is reflected. Skip only if a load is already in-flight.
    if (get().loading) return;
    set({ loading: true, error: null });
    try {
      const msgs = await fetchHistory(50);
      set({ messages: msgs, loaded: true });
    } catch (e: any) {
      set({ error: e.message });
    } finally {
      set({ loading: false });
    }
  },

  async send(text: string) {
    const trimmed = text.trim();
    if (!trimmed || get().sending) return;
    set({ sending: true, error: null });

    // Optimistic user message so the transcript updates instantly.
    const optimistic: ChatMessage = {
      id: `optimistic-${Date.now()}`,
      userId: 'self',
      role: 'user',
      content: trimmed,
      createdAt: new Date().toISOString(),
    };
    set((s) => ({ messages: [...s.messages, optimistic] }));

    try {
      const res = await sendMessage(trimmed);
      const serverMsgs = res.messages ?? [];
      set((s) => {
        // Drop the optimistic message, append the two server-persisted ones.
        const filtered = s.messages.filter((m) => m.id !== optimistic.id);
        return {
          messages: [...filtered, ...serverMsgs],
          lastToolCalls: res.toolCalls ?? [],
        };
      });

      // Refresh todos if Marvin mutated them.
      if ((res.toolCalls ?? []).some((c) => TODO_MUTATING_TOOLS.has(c.name))) {
        useTodoStore.getState().fetchTodos();
      }
    } catch (e: any) {
      set((s) => ({
        error: e.message,
        messages: s.messages.filter((m) => m.id !== optimistic.id),
      }));
    } finally {
      set({ sending: false });
    }
  },

  async reset() {
    try {
      await clearHistory();
      set({ messages: [], lastToolCalls: [] });
    } catch (e: any) {
      set({ error: e.message });
    }
  },
}));
