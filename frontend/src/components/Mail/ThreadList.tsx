import { ScrollArea, Stack, UnstyledButton, Group, Text, Badge } from '@mantine/core';
import dayjs from 'dayjs';
import type { MailThread } from '../../types/mail';

interface Props {
  threads: MailThread[];
  activeThreadId: string | null;
  onSelect: (threadId: string) => void;
}

export function ThreadList({ threads, activeThreadId, onSelect }: Props) {
  if (threads.length === 0) {
    return (
      <Text c="dimmed" size="sm" ta="center" py="lg">
        Inbox zero. *whirrrr*
      </Text>
    );
  }
  return (
    <ScrollArea style={{ flex: 1 }} type="auto">
      <Stack gap={4}>
        {threads.map((t) => {
          const active = t.threadId === activeThreadId;
          return (
            <UnstyledButton
              key={t.threadId}
              onClick={() => onSelect(t.threadId)}
              style={{
                padding: '8px 10px',
                borderRadius: 8,
                background: active ? 'var(--mantine-color-synthPurple-light)' : 'transparent',
                borderLeft: active ? '3px solid var(--mantine-color-synthPurple-6)' : '3px solid transparent',
              }}
            >
              <Group justify="space-between" gap={6} wrap="nowrap">
                <Text size="sm" fw={t.unread > 0 ? 700 : 500} truncate>
                  {t.participantName}
                </Text>
                <Group gap={4} wrap="nowrap">
                  {t.unread > 0 && (
                    <Badge size="xs" color="neonPink" circle>
                      {t.unread}
                    </Badge>
                  )}
                  <Text size="10px" c="dimmed">
                    {dayjs(t.lastMessageAt).format('MMM D')}
                  </Text>
                </Group>
              </Group>
              <Text size="xs" fw={t.unread > 0 ? 600 : 400} truncate>
                {t.subject}
              </Text>
              <Text size="11px" c="dimmed" truncate>
                {t.snippet}
              </Text>
            </UnstyledButton>
          );
        })}
      </Stack>
    </ScrollArea>
  );
}
