import { create } from 'zustand';
import * as api from '../api/contacts';
import type { Contact, ContactCreate } from '../types/contact';

interface ContactState {
  contacts: Contact[];
  loading: boolean;
  fetch: () => Promise<void>;
  add: (data: ContactCreate) => Promise<void>;
  update: (id: string, data: Partial<ContactCreate>) => Promise<void>;
  remove: (id: string) => Promise<void>;
}

export const useContactStore = create<ContactState>((set, get) => ({
  contacts: [],
  loading: false,

  fetch: async () => {
    set({ loading: true });
    try {
      set({ contacts: await api.fetchContacts() });
    } finally {
      set({ loading: false });
    }
  },

  add: async (data) => {
    const c = await api.createContact(data);
    set((s) => ({ contacts: [...s.contacts, c].sort((a, b) => a.name.localeCompare(b.name)) }));
  },

  update: async (id, data) => {
    const updated = await api.updateContact(id, data);
    set((s) => ({ contacts: s.contacts.map((c) => (c.id === id ? updated : c)) }));
  },

  remove: async (id) => {
    const prev = get().contacts;
    set((s) => ({ contacts: s.contacts.filter((c) => c.id !== id) }));
    try {
      await api.deleteContact(id);
    } catch (e) {
      set({ contacts: prev });
      throw e;
    }
  },
}));
