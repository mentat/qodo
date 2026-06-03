import { create } from 'zustand';
import { collection, onSnapshot, orderBy, query, where } from 'firebase/firestore';
import { db } from '../firebase';
import type { CalendarEvent } from '../types/event';
import * as api from '../api/events';

function toDate(v: unknown): Date {
  if (v instanceof Date) return v;
  if (typeof v === 'string') return new Date(v);
  if (v && typeof v === 'object' && 'toDate' in v && typeof (v as { toDate: unknown }).toDate === 'function') {
    return (v as { toDate(): Date }).toDate();
  }
  return new Date();
}

function normalize(id: string, d: Record<string, unknown>): CalendarEvent {
  const start = toDate(d.start);
  let end = toDate(d.end);
  if (end <= start) end = new Date(start.getTime() + 60 * 60 * 1000);
  return {
    id,
    userId: String(d.userId ?? ''),
    title: String(d.title ?? ''),
    description: String(d.description ?? ''),
    location: String(d.location ?? ''),
    start,
    end,
    allDay: Boolean(d.allDay),
    color: String(d.color ?? '#9B5DE5'),
    characterId: d.characterId ? String(d.characterId) : undefined,
    kind: 'event',
  };
}

interface EventState {
  events: CalendarEvent[];
  loading: boolean;
  unsub: (() => void) | null;
  subscribe: (uid: string) => void;
  unsubscribe: () => void;
  create: (input: api.EventInput) => Promise<void>;
  update: (id: string, input: api.EventInput) => Promise<void>;
  move: (id: string, start: Date, end?: Date) => Promise<void>;
  remove: (id: string) => Promise<void>;
}

export const useEventStore = create<EventState>((set, get) => ({
  events: [],
  loading: true,
  unsub: null,

  subscribe: (uid) => {
    if (get().unsub) return;
    const q = query(collection(db, 'events'), where('userId', '==', uid), orderBy('start', 'asc'));
    const unsub = onSnapshot(
      q,
      (snap) => {
        const events = snap.docs.map((doc) => normalize(doc.id, doc.data() as Record<string, unknown>));
        set({ events, loading: false });
      },
      (err) => {
        console.error('events snapshot error', err);
        set({ loading: false });
      },
    );
    set({ unsub });
  },

  unsubscribe: () => {
    const u = get().unsub;
    if (u) u();
    set({ unsub: null, events: [], loading: true });
  },

  create: async (input) => {
    await api.createEvent(input);
  },

  update: async (id, input) => {
    await api.updateEvent(id, input);
  },

  move: async (id, start, end) => {
    await api.moveEvent(id, start.toISOString(), end?.toISOString());
  },

  remove: async (id) => {
    await api.deleteEvent(id);
  },
}));
