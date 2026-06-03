import { describe, it, expect } from 'bun:test';
import { deriveThreads, totalUnread } from '../src/store/mailStore';
import type { Email } from '../src/types/mail';

function email(over: Partial<Email>): Email {
  return {
    id: 'x',
    userId: 'u',
    threadId: 't1',
    from: 'a@b',
    fromName: 'A',
    to: 'me',
    subject: 'S',
    body: 'B',
    direction: 'inbound',
    read: false,
    createdAt: '2026-01-01T00:00:00Z',
    ...over,
  };
}

describe('deriveThreads', () => {
  it('groups by threadId, newest thread first, messages oldest-first', () => {
    const emails: Email[] = [
      email({ id: '1', threadId: 't1', subject: 'Hi', fromName: 'Dot', createdAt: '2026-01-01T00:00:00Z' }),
      email({ id: '2', threadId: 't1', direction: 'outbound', from: 'me', to: 'dot@x', fromName: 'You', createdAt: '2026-01-01T01:00:00Z' }),
      email({ id: '3', threadId: 't2', subject: 'Yo', fromName: 'Reg', createdAt: '2026-01-02T00:00:00Z' }),
    ];
    const threads = deriveThreads(emails);
    expect(threads.length).toBe(2);
    expect(threads[0].threadId).toBe('t2'); // most recent lastMessageAt
    const t1 = threads.find((t) => t.threadId === 't1');
    expect(t1?.messages.length).toBe(2);
    expect(t1?.subject).toBe('Hi');
    expect(t1?.participantName).toBe('Dot');
    expect(t1?.messages[0].id).toBe('1'); // oldest first
  });

  it('counts unread inbound messages per thread', () => {
    const emails: Email[] = [
      email({ id: '1', threadId: 'a', direction: 'inbound', read: false }),
      email({ id: '2', threadId: 'a', direction: 'inbound', read: false }),
      email({ id: '3', threadId: 'a', direction: 'outbound', read: false }),
    ];
    const [t] = deriveThreads(emails);
    expect(t.unread).toBe(2);
  });
});

describe('totalUnread', () => {
  it('counts only unread inbound mail', () => {
    const emails: Email[] = [
      email({ id: '1', direction: 'inbound', read: false }),
      email({ id: '2', direction: 'inbound', read: true }),
      email({ id: '3', direction: 'outbound', read: false }),
    ];
    expect(totalUnread(emails)).toBe(1);
  });
});
