import { ScrollArea, Stack, Group, Text, Badge, Avatar, ActionIcon, Tooltip } from '@mantine/core';
import { IconStar, IconStarFilled, IconPaperclip } from '@tabler/icons-react';
import type { MailThread } from '../../types/mail';
import { initials, senderColor, relativeTime, exactTime } from './mailUi';

interface Props {
  threads: MailThread[];
  activeThreadId: string | null;
  onSelect: (threadId: string) => void;
  onStar: (emailId: string, starred: boolean) => void;
}

export function ThreadList({ threads, activeThreadId, onSelect, onStar }: Props) {
  if (threads.length === 0) {
    return (
      <Text c="dimmed" size="sm" ta="center" py="lg">
        Nothing here. *whirrrr*
      </Text>
    );
  }
  return (
    <ScrollArea style={{ flex: 1 }} type="auto">
      <Stack gap={2}>
        {threads.map((t) => {
          const active = t.threadId === activeThreadId;
          const last = t.messages[t.messages.length - 1];
          return (
            <div
              key={t.threadId}
              role="button"
              tabIndex={0}
              onClick={() => onSelect(t.threadId)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  onSelect(t.threadId);
                }
              }}
              style={{
                cursor: 'pointer',
                padding: '8px 10px',
                borderRadius: 8,
                background: active ? 'var(--mantine-color-synthPurple-light)' : 'transparent',
                borderLeft: active ? '3px solid var(--mantine-color-synthPurple-6)' : '3px solid transparent',
              }}
            >
              <Group gap={8} wrap="nowrap" align="flex-start">
                <Avatar size={34} radius="xl" color={senderColor(t.characterId || t.participantName)}>
                  {initials(t.participantName)}
                </Avatar>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Group justify="space-between" gap={6} wrap="nowrap">
                    <Text size="sm" fw={t.unread > 0 ? 700 : 500} truncate>
                      {t.participantName}
                    </Text>
                    <Tooltip label={exactTime(t.lastMessageAt)} openDelay={400} withinPortal>
                      <Text size="10px" c="dimmed" style={{ whiteSpace: 'nowrap' }}>
                        {relativeTime(t.lastMessageAt)}
                      </Text>
                    </Tooltip>
                  </Group>
                  <Group justify="space-between" gap={6} wrap="nowrap">
                    <Text size="xs" fw={t.unread > 0 ? 600 : 400} truncate>
                      {t.subject}
                    </Text>
                    <Group gap={4} wrap="nowrap">
                      {t.hasAttachment && <IconPaperclip size={12} style={{ opacity: 0.6 }} />}
                      {t.unread > 0 && (
                        <Badge size="xs" color="neonPink" circle>
                          {t.unread}
                        </Badge>
                      )}
                    </Group>
                  </Group>
                  <Group justify="space-between" gap={6} wrap="nowrap">
                    <Text size="11px" c="dimmed" truncate>
                      {t.snippet}
                    </Text>
                    <ActionIcon
                      variant="subtle"
                      size="sm"
                      color={t.starred ? 'hotYellow' : 'gray'}
                      onClick={(e) => {
                        e.stopPropagation();
                        onStar(last.id, !t.starred);
                      }}
                      aria-label={t.starred ? 'Unstar thread' : 'Star thread'}
                    >
                      {t.starred ? <IconStarFilled size={14} /> : <IconStar size={14} />}
                    </ActionIcon>
                  </Group>
                </div>
              </Group>
            </div>
          );
        })}
      </Stack>
    </ScrollArea>
  );
}
