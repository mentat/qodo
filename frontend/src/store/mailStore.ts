import { create } from 'zustand';
import { collection, limit, onSnapshot, orderBy, query, where } from 'firebase/firestore';
import { db } from '../firebase';
import type { Email, MailThread } from '../types/mail';
import * as api from '../api/mail';

function toISO(v: unknown): string {
  if (typeof v === 'string') return v;
  if (v instanceof Date) return v.toISOString();
  if (v && typeof v === 'object' && 'toDate' in v && typeof (v as { toDate: unknown }).toDate === 'function') {
    return (v as { toDate(): Date }).toDate().toISOString();
  }
  return new Date(0).toISOString();
}

function normalize(id: string, d: Record<string, unknown>): Email {
  return {
    id,
    userId: String(d.userId ?? ''),
    threadId: String(d.threadId ?? ''),
    from: String(d.from ?? ''),
    fromName: String(d.fromName ?? ''),
    to: String(d.to ?? ''),
    subject: String(d.subject ?? ''),
    body: String(d.body ?? ''),
    direction: d.direction === 'outbound' ? 'outbound' : 'inbound',
    read: Boolean(d.read),
    characterId: d.characterId ? String(d.characterId) : undefined,
    createdAt: toISO(d.createdAt),
  };
}

interface MailState {
  emails: Email[];
  loading: boolean;
  activeThreadId: string | null;
  unsub: (() => void) | null;
  subscribe: (uid: string) => void;
  unsubscribe: () => void;
  selectThread: (threadId: string | null) => void;
  send: (input: api.SendInput) => Promise<void>;
  markRead: (id: string) => Promise<void>;
  remove: (id: string) => Promise<void>;
}

export const useMailStore = create<MailState>((set, get) => ({
  emails: [],
  loading: true,
  activeThreadId: null,
  unsub: null,

  subscribe: (uid) => {
    if (get().unsub) return; // already live
    const q = query(
      collection(db, 'emails'),
      where('userId', '==', uid),
      orderBy('createdAt', 'desc'),
      limit(200),
    );
    const unsub = onSnapshot(
      q,
      (snap) => {
        const emails = snap.docs.map((doc) => normalize(doc.id, doc.data() as Record<string, unknown>));
        set({ emails, loading: false });
      },
      (err) => {
        console.error('mail snapshot error', err);
        set({ loading: false });
      },
    );
    set({ unsub });
  },

  unsubscribe: () => {
    const u = get().unsub;
    if (u) u();
    set({ unsub: null, emails: [], loading: true, activeThreadId: null });
  },

  selectThread: (threadId) => {
    set({ activeThreadId: threadId });
    if (!threadId) return;
    // Mark unread inbound messages in this thread as read (best-effort).
    for (const e of get().emails) {
      if (e.threadId === threadId && e.direction === 'inbound' && !e.read) {
        void api.markEmailRead(e.id).catch(() => undefined);
      }
    }
  },

  send: async (input) => {
    // Writes go through the API (so the server publishes the reply job); the
    // onSnapshot listener reflects the new message moments later.
    await api.sendEmail(input);
  },

  markRead: async (id) => {
    await api.markEmailRead(id);
  },

  remove: async (id) => {
    await api.deleteEmail(id);
  },
}));

// deriveThreads groups the flat email list into threads for the list view.
export function deriveThreads(emails: Email[]): MailThread[] {
  const byThread = new Map<string, Email[]>();
  for (const e of emails) {
    const arr = byThread.get(e.threadId) ?? [];
    arr.push(e);
    byThread.set(e.threadId, arr);
  }
  const threads: MailThread[] = [];
  byThread.forEach((msgs, threadId) => {
    msgs.sort((a, b) => a.createdAt.localeCompare(b.createdAt));
    const last = msgs[msgs.length - 1];
    const inbound = msgs.find((m) => m.direction === 'inbound');
    const participantName = inbound?.fromName || last.to || last.fromName || 'Unknown';
    const characterId = msgs.find((m) => m.characterId)?.characterId;
    const unread = msgs.filter((m) => m.direction === 'inbound' && !m.read).length;
    threads.push({
      threadId,
      subject: msgs[0].subject || '(no subject)',
      characterId,
      participantName,
      lastMessageAt: last.createdAt,
      snippet: last.body.replace(/\s+/g, ' ').slice(0, 100),
      unread,
      messages: msgs,
    });
  });
  threads.sort((a, b) => b.lastMessageAt.localeCompare(a.lastMessageAt));
  return threads;
}

export function totalUnread(emails: Email[]): number {
  return emails.filter((e) => e.direction === 'inbound' && !e.read).length;
}
