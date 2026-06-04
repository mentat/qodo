import { describe, it, expect } from 'bun:test';
import { deriveThreads, filterThreads } from '../src/store/mailStore';
import type { Email } from '../src/types/mail';

function email(over: Partial<Email>): Email {
  return {
    id: 'x',
    userId: 'u',
    threadId: 't1',
    from: 'a@b',
    fromName: 'Alice',
    to: 'me',
    subject: 'Hello',
    body: 'body text',
    direction: 'inbound',
    read: false,
    starred: false,
    createdAt: '2026-01-01T00:00:00Z',
    ...over,
  };
}

const emails: Email[] = [
  email({ id: '1', threadId: 't1', fromName: 'Dot Matrix', subject: 'Toner', body: 'paper jam', read: false, starred: true, createdAt: '2026-01-03T00:00:00Z' }),
  email({
    id: '2',
    threadId: 't2',
    fromName: 'Capt Nimbus',
    subject: 'Migration',
    body: 'cloud weather',
    read: true,
    attachments: [{ name: 'plan.xlsx', size: 10, contentType: 'application/vnd.ms-excel' }],
    createdAt: '2026-01-02T00:00:00Z',
  }),
  email({ id: '3', threadId: 't3', fromName: 'Brad', subject: 'Sync', body: 'synergies', read: true, createdAt: '2026-01-01T00:00:00Z' }),
];
const threads = deriveThreads(emails);

describe('filterThreads', () => {
  it('returns everything for all + empty query', () => {
    expect(filterThreads(threads, '', 'all').length).toBe(3);
  });

  it('unread filter keeps only threads with unread inbound', () => {
    expect(filterThreads(threads, '', 'unread').map((t) => t.threadId)).toEqual(['t1']);
  });

  it('starred filter keeps only starred threads', () => {
    expect(filterThreads(threads, '', 'starred').map((t) => t.threadId)).toEqual(['t1']);
  });

  it('attachments filter keeps only threads with an attachment', () => {
    expect(filterThreads(threads, '', 'attachments').map((t) => t.threadId)).toEqual(['t2']);
  });

  it('text query matches subject, body, and participant', () => {
    expect(filterThreads(threads, 'synergies', 'all').map((t) => t.threadId)).toEqual(['t3']);
    expect(filterThreads(threads, 'nimbus', 'all').map((t) => t.threadId)).toEqual(['t2']);
    expect(filterThreads(threads, 'toner', 'all').map((t) => t.threadId)).toEqual(['t1']);
  });

  it('combines query + filter', () => {
    expect(filterThreads(threads, 'paper', 'starred').length).toBe(1);
    expect(filterThreads(threads, 'synergies', 'starred').length).toBe(0);
  });
});
