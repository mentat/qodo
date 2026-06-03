import { create } from 'zustand';
import * as api from '../api/notes';
import type { Note, NoteCreate } from '../types/note';

interface NoteState {
  notes: Note[];
  loading: boolean;
  fetch: () => Promise<void>;
  add: (data: NoteCreate) => Promise<Note>;
  update: (id: string, data: NoteCreate) => Promise<void>;
  remove: (id: string) => Promise<void>;
}

export const useNoteStore = create<NoteState>((set, get) => ({
  notes: [],
  loading: false,

  fetch: async () => {
    set({ loading: true });
    try {
      set({ notes: await api.fetchNotes() });
    } finally {
      set({ loading: false });
    }
  },

  add: async (data) => {
    const n = await api.createNote(data);
    set((s) => ({ notes: [n, ...s.notes] }));
    return n;
  },

  update: async (id, data) => {
    const updated = await api.updateNote(id, data);
    set((s) => ({ notes: s.notes.map((n) => (n.id === id ? updated : n)).sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)) }));
  },

  remove: async (id) => {
    const prev = get().notes;
    set((s) => ({ notes: s.notes.filter((n) => n.id !== id) }));
    try {
      await api.deleteNote(id);
    } catch (e) {
      set({ notes: prev });
      throw e;
    }
  },
}));
