export interface Attachment {
  name: string;
  size: number; // bytes
  contentType: string;
}

export interface Email {
  id: string;
  userId: string;
  threadId: string;
  from: string;
  fromName: string;
  to: string;
  cc?: string[];
  subject: string;
  body: string;
  signature?: string;
  attachments?: Attachment[];
  direction: 'inbound' | 'outbound';
  read: boolean;
  starred: boolean;
  characterId?: string;
  createdAt: string;
}

// MailThread is derived client-side by grouping emails on threadId.
export interface MailThread {
  threadId: string;
  subject: string;
  characterId?: string;
  participantName: string;
  lastMessageAt: string;
  snippet: string;
  unread: number;
  starred: boolean;
  hasAttachment: boolean;
  messages: Email[];
}

// MailFilter drives the inbox quick-filter chips.
export type MailFilter = 'all' | 'unread' | 'starred' | 'attachments';
