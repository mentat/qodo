import { useState } from 'react';
import { Stack, Group, TextInput, Textarea, Button } from '@mantine/core';
import { IconSend, IconX } from '@tabler/icons-react';
import type { MailThread } from '../../types/mail';
import type { SendInput } from '../../api/mail';

interface Props {
  mode: 'reply' | 'new';
  thread?: MailThread;
  sending: boolean;
  onSend: (input: SendInput) => void;
  onCancel?: () => void;
}

export function Composer({ mode, thread, sending, onSend, onCancel }: Props) {
  const [to, setTo] = useState('');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');

  const submit = () => {
    if (!body.trim() || sending) return;
    if (mode === 'reply' && thread) {
      const inbound = thread.messages.find((m) => m.direction === 'inbound');
      const replyTo = inbound?.from ?? thread.messages[0]?.to ?? '';
      const subj = thread.subject.toLowerCase().startsWith('re:') ? thread.subject : `Re: ${thread.subject}`;
      onSend({ to: replyTo, subject: subj, body, threadId: thread.threadId, characterId: thread.characterId });
    } else {
      if (!to.trim()) return;
      onSend({ to, subject: subject || '(no subject)', body });
    }
    setBody('');
    setSubject('');
    setTo('');
  };

  return (
    <Stack gap={6} pt="sm" style={{ borderTop: '1px solid var(--mantine-color-default-border)' }}>
      {mode === 'new' && (
        <>
          <TextInput
            placeholder="To (e.g. Capt. Nimbus or nimbus@synthwave.os)"
            value={to}
            onChange={(e) => setTo(e.currentTarget.value)}
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
