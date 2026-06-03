import { useMemo, useState } from 'react';
import { Grid, Group, Button, Paper, Text, Stack } from '@mantine/core';
import { IconPencilPlus, IconMail } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { useMailStore, deriveThreads } from '../../store/mailStore';
import { ThreadList } from './ThreadList';
import { ThreadView } from './ThreadView';
import { Composer } from './Composer';
import type { SendInput } from '../../api/mail';

export function MailApp() {
  const emails = useMailStore((s) => s.emails);
  const activeThreadId = useMailStore((s) => s.activeThreadId);
  const selectThread = useMailStore((s) => s.selectThread);
  const send = useMailStore((s) => s.send);
  const [composing, setComposing] = useState(false);
  const [sending, setSending] = useState(false);

  const threads = useMemo(() => deriveThreads(emails), [emails]);
  const activeThread = useMemo(
    () => threads.find((t) => t.threadId === activeThreadId) ?? null,
    [threads, activeThreadId],
  );

  const doSend = async (input: SendInput) => {
    setSending(true);
    try {
      await send(input);
      setComposing(false);
      notifications.show({ title: 'Sent', message: 'Message away ✦', color: 'neonGreen' });
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    } finally {
      setSending(false);
    }
  };

  return (
    <Grid gutter="md" style={{ height: 'calc(100vh - 92px)' }}>
      <Grid.Col span={{ base: 12, sm: 4 }} style={{ height: '100%' }}>
        <Stack gap="xs" style={{ height: '100%' }}>
          <Button
            leftSection={<IconPencilPlus size={16} />}
            onClick={() => {
              setComposing(true);
              selectThread(null);
            }}
            variant="light"
          >
            Compose
          </Button>
          <ThreadList
            threads={threads}
            activeThreadId={activeThreadId}
            onSelect={(id) => {
              setComposing(false);
              selectThread(id);
            }}
          />
        </Stack>
      </Grid.Col>

      <Grid.Col span={{ base: 12, sm: 8 }} style={{ height: '100%' }}>
        <Paper withBorder radius="md" p="md" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          {composing ? (
            <Composer mode="new" sending={sending} onSend={doSend} onCancel={() => setComposing(false)} />
          ) : activeThread ? (
            <>
              <ThreadView thread={activeThread} />
              <Composer mode="reply" thread={activeThread} sending={sending} onSend={doSend} />
            </>
          ) : (
            <Group justify="center" align="center" style={{ flex: 1 }}>
              <Stack align="center" gap={4}>
                <IconMail size={40} opacity={0.4} />
                <Text c="dimmed" size="sm">
                  Select a thread or compose a new message.
                </Text>
              </Stack>
            </Group>
          )}
        </Paper>
      </Grid.Col>
    </Grid>
  );
}
