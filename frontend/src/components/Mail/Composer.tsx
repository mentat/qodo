import { useMemo, useState } from 'react';
import { Autocomplete, Button, Group, Stack, Text, Textarea, TextInput } from '@mantine/core';
import { IconSend, IconX } from '@tabler/icons-react';
import type { MailThread } from '../../types/mail';
import type { SendInput } from '../../api/mail';
import type { Contact } from '../../types/contact';

interface RecipientOption {
  value: string;
  label: string;
}

interface RecipientData {
  options: RecipientOption[];
  byKey: Map<string, Contact>;
}

interface Props {
  mode: 'reply' | 'new';
  thread?: MailThread;
  contacts?: Contact[];
  sending: boolean;
  onSend: (input: SendInput) => void;
  onCancel?: () => void;
}

function recipientLabel(contact: Contact) {
  const email = contact.email.trim();
  const name = contact.name.trim();
  return name ? `${name} <${email}>` : email;
}

function recipientKey(value: string) {
  return value.trim().toLowerCase();
}

function buildRecipientData(contacts: Contact[]): RecipientData {
  const seen = new Set<string>();
  const byKey = new Map<string, Contact>();
  const options: RecipientOption[] = [];

  for (const contact of contacts) {
    const email = contact.email.trim();
    if (!email) continue;

    const emailKey = recipientKey(email);
    if (seen.has(emailKey)) continue;
    seen.add(emailKey);

    const label = recipientLabel(contact);
    options.push({ value: email, label });
    byKey.set(emailKey, contact);
    byKey.set(recipientKey(label), contact);
  }

  return { options, byKey };
}

export function Composer({ mode, thread, contacts = [], sending, onSend, onCancel }: Props) {
  const [to, setTo] = useState('');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const recipients = useMemo(() => buildRecipientData(contacts), [contacts]);

  const submit = () => {
    if (!body.trim() || sending) return;
    if (mode === 'reply' && thread) {
      const inbound = thread.messages.find((m) => m.direction === 'inbound');
      const replyTo = inbound?.from ?? thread.messages[0]?.to ?? '';
      const subj = thread.subject.toLowerCase().startsWith('re:') ? thread.subject : `Re: ${thread.subject}`;
      onSend({ to: replyTo, subject: subj, body, threadId: thread.threadId, characterId: thread.characterId });
    } else {
      const recipient = recipients.byKey.get(recipientKey(to));
      const recipientAddress = (recipient?.email ?? to).trim();
      if (!recipientAddress) return;
      onSend({
        to: recipientAddress,
        toName: recipient?.name || undefined,
        subject: subject || '(no subject)',
        body,
        characterId: recipient?.characterId,
      });
    }
    setBody('');
    setSubject('');
    setTo('');
  };

  return (
    <Stack gap={6} pt="sm" style={{ borderTop: '1px solid var(--mantine-color-default-border)' }}>
      {mode === 'new' && (
        <>
          <Autocomplete
            placeholder="To"
            data={recipients.options}
            value={to}
            onChange={setTo}
            limit={8}
            openOnFocus
            renderOption={({ option }) => {
              const contact = recipients.byKey.get(recipientKey(option.value));
              return (
                <Group gap={8} wrap="nowrap">
                  <div style={{ minWidth: 0 }}>
                    <Text size="sm" fw={600} truncate>
                      {contact?.name || option.value}
                    </Text>
                    <Text size="xs" c="dimmed" truncate>
                      {contact?.company ? `${option.value} · ${contact.company}` : option.value}
                    </Text>
                  </div>
                </Group>
              );
            }}
          />
          <TextInput
            placeholder="Subject"
            value={subject}
            onChange={(e) => setSubject(e.currentTarget.value)}
          />
        </>
      )}
      <Textarea
        placeholder={mode === 'reply' ? 'Write your reply…' : 'Write your message…'}
        autosize
        minRows={2}
        maxRows={6}
        value={body}
        onChange={(e) => setBody(e.currentTarget.value)}
      />
      <Group justify="flex-end" gap={6}>
        {onCancel && (
          <Button variant="subtle" color="gray" leftSection={<IconX size={14} />} onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button
          leftSection={<IconSend size={14} />}
          onClick={submit}
          loading={sending}
          disabled={!body.trim() || (mode === 'new' && !to.trim())}
        >
          Send
        </Button>
      </Group>
    </Stack>
  );
}
