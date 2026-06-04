import { ScrollArea, Stack, Box, Text, Group, Avatar, ActionIcon, Tooltip, Badge } from '@mantine/core';
import { IconStar, IconStarFilled } from '@tabler/icons-react';
import type { MailThread, Email } from '../../types/mail';
import { initials, senderColor, relativeTime, exactTime, attachmentIcon, formatBytes } from './mailUi';

interface Props {
  thread: MailThread;
  onStar: (id: string, starred: boolean) => void;
}

export function ThreadView({ thread, onStar }: Props) {
  const last = thread.messages[thread.messages.length - 1];
  const awaitingReply = last?.direction === 'outbound' && !!thread.characterId;

  return (
    <>
      <Text fw={700} size="lg" mb={8} lineClamp={1}>
        {thread.subject}
      </Text>
      <ScrollArea style={{ flex: 1 }} type="auto">
        <Stack gap={12} pb="sm">
          {thread.messages.map((m) => (
            <MessageCard key={m.id} m={m} onStar={onStar} />
          ))}
          {awaitingReply && (
            <Group gap={8}>
              <Avatar size={28} radius="xl" color={senderColor(thread.characterId || thread.participantName)}>
                {initials(thread.participantName)}
              </Avatar>
              <Text size="xs" c="dimmed" fs="italic">
                {thread.participantName} is composing a reply… ✦
              </Text>
            </Group>
          )}
        </Stack>
      </ScrollArea>
    </>
  );
}

function MessageCard({ m, onStar }: { m: Email; onStar: (id: string, starred: boolean) => void }) {
  const mine = m.direction === 'outbound';
  const who = mine ? 'You' : m.fromName || m.from;
  const toLine = mine ? `to ${m.to || 'them'}` : 'to you';
  const ccLine = m.cc?.length ? ` · cc ${m.cc.join(', ')}` : '';

  return (
    <Box
      style={{
        border: '1px solid var(--mantine-color-default-border)',
        borderRadius: 10,
        padding: 12,
        background: 'var(--mantine-color-body)',
      }}
    >
      <Group justify="space-between" wrap="nowrap" align="flex-start" mb={6}>
        <Group gap={8} wrap="nowrap">
          <Avatar size={34} radius="xl" color={senderColor(mine ? 'me' : m.characterId || m.from)}>
            {mine ? 'ME' : initials(who)}
          </Avatar>
          <div style={{ minWidth: 0 }}>
            <Text size="sm" fw={700}>
              {who}
            </Text>
            <Text size="11px" c="dimmed" lineClamp={1}>
              {toLine}
              {ccLine}
            </Text>
          </div>
        </Group>
        <Group gap={4} wrap="nowrap">
          <Tooltip label={exactTime(m.createdAt)} openDelay={300} withinPortal>
            <Text size="10px" c="dimmed" style={{ whiteSpace: 'nowrap' }}>
              {relativeTime(m.createdAt)}
            </Text>
          </Tooltip>
          <ActionIcon
            variant="subtle"
            size="sm"
            color={m.starred ? 'hotYellow' : 'gray'}
            onClick={() => onStar(m.id, !m.starred)}
            aria-label={m.starred ? 'Unstar' : 'Star'}
          >
            {m.starred ? <IconStarFilled size={14} /> : <IconStar size={14} />}
          </ActionIcon>
        </Group>
      </Group>

      <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
        {m.body}
      </Text>

      {m.signature && (
        <Text
          size="xs"
          c="dimmed"
          mt={8}
          style={{ whiteSpace: 'pre-wrap', borderTop: '1px dashed var(--mantine-color-default-border)', paddingTop: 6 }}
        >
          {m.signature}
        </Text>
      )}

      {!!m.attachments?.length && (
        <Group gap={6} mt={8}>
          {m.attachments.map((a) => (
            <Tooltip key={a.name} label="Mocked attachment (demo)" openDelay={200} withinPortal>
              <Badge
                variant="light"
                color="synthPurple"
                leftSection={<span aria-hidden>{attachmentIcon(a.contentType)}</span>}
                style={{ textTransform: 'none', cursor: 'default', maxWidth: 240 }}
              >
                {a.name} · {formatBytes(a.size)}
              </Badge>
            </Tooltip>
          ))}
        </Group>
      )}
    </Box>
  );
}
