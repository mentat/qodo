export interface Email {
  id: string;
  userId: string;
  threadId: string;
  from: string;
  fromName: string;
  to: string;
  subject: string;
  body: string;
  direction: 'inbound' | 'outbound';
  read: boolean;
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
  messages: Email[];
}
