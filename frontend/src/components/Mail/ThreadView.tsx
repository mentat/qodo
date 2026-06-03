import { ScrollArea, Stack, Box, Text, Group } from '@mantine/core';
import dayjs from 'dayjs';
import type { MailThread } from '../../types/mail';

export function ThreadView({ thread }: { thread: MailThread }) {
  const last = thread.messages[thread.messages.length - 1];
  const awaitingReply = last?.direction === 'outbound' && !!thread.characterId;

  return (
    <>
      <Text fw={700} size="lg" mb={4} lineClamp={1}>
        {thread.subject}
      </Text>
      <ScrollArea style={{ flex: 1 }} type="auto">
        <Stack gap={10} pb="sm">
          {thread.messages.map((m) => {
            const mine = m.direction === 'outbound';
            return (
              <Box
                key={m.id}
                style={{
                  alignSelf: mine ? 'flex-end' : 'flex-start',
                  maxWidth: '82%',
                  background: mine ? 'var(--mantine-color-synthPurple-light)' : 'var(--mantine-color-electricBlue-light)',
                  borderRadius: 10,
                  padding: '8px 12px',
                }}
              >
                <Group gap={6} mb={2}>
                  <Text size="xs" fw={700}>
                    {mine ? 'You' : m.fromName || m.from}
                  </Text>
                  <Text size="10px" c="dimmed">
                    {dayjs(m.createdAt).format('MMM D, h:mm A')}
                  </Text>
                </Group>
                <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
                  {m.body}
                </Text>
              </Box>
            );
          })}
          {awaitingReply && (
            <Box style={{ alignSelf: 'flex-start' }}>
              <Text size="xs" c="dimmed" fs="italic">
                {thread.participantName} is composing a reply… ✦
              </Text>
            </Box>
          )}
        </Stack>
      </ScrollArea>
    </>
  );
}
