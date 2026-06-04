import { useEffect, useMemo, useState } from 'react';
import { Box, Group, Button, Paper, Text, Stack, TextInput, SegmentedControl } from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import { IconPencilPlus, IconMail, IconSearch } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { useMailStore, deriveThreads, filterThreads } from '../../store/mailStore';
import { useContactStore } from '../../store/contactStore';
import type { MailFilter } from '../../types/mail';
import { ThreadList } from './ThreadList';
import { ThreadView } from './ThreadView';
import { Composer } from './Composer';
import type { SendInput } from '../../api/mail';

export function MailApp() {
  const emails = useMailStore((s) => s.emails);
  const activeThreadId = useMailStore((s) => s.activeThreadId);
  const selectThread = useMailStore((s) => s.selectThread);
  const send = useMailStore((s) => s.send);
  const toggleStar = useMailStore((s) => s.toggleStar);
  const contacts = useContactStore((s) => s.contacts);
  const fetchContacts = useContactStore((s) => s.fetch);
  const [composing, setComposing] = useState(false);
  const [sending, setSending] = useState(false);
  const [search, setSearch] = useState('');
  const [debounced] = useDebouncedValue(search, 200);
  const [filter, setFilter] = useState<MailFilter>('all');

  const threads = useMemo(() => deriveThreads(emails), [emails]);
  const visible = useMemo(() => filterThreads(threads, debounced, filter), [threads, debounced, filter]);
  const activeThread = useMemo(
    () => threads.find((t) => t.threadId === activeThreadId) ?? null,
    [threads, activeThreadId],
  );

  useEffect(() => {
    void fetchContacts().catch((e) =>
      notifications.show({ title: 'Contacts unavailable', message: (e as Error).message, color: 'red' }),
    );
  }, [fetchContacts]);

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

  const onStar = (id: string, starred: boolean) => {
    void toggleStar(id, starred).catch((e) =>
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' }),
    );
  };

  return (
    <Box style={{ display: 'flex', gap: 'var(--mantine-spacing-md)', height: 'calc(100vh - 92px)' }}>
      <Box style={{ flex: '0 0 340px', minWidth: 0, display: 'flex', flexDirection: 'column', height: '100%' }}>
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
        <TextInput
          mt="xs"
          placeholder="Search mail…"
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
        />
        <SegmentedControl
          mt="xs"
          fullWidth
          size="xs"
          value={filter}
          onChange={(v) => setFilter(v as MailFilter)}
          data={[
            { label: 'All', value: 'all' },
            { label: 'Unread', value: 'unread' },
            { label: '★', value: 'starred' },
            { label: '📎', value: 'attachments' },
          ]}
        />
        <Box mt="xs" style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
          <ThreadList
            threads={visible}
            activeThreadId={activeThreadId}
            onSelect={(id) => {
              setComposing(false);
              selectThread(id);
            }}
            onStar={onStar}
          />
        </Box>
      </Box>

      <Box style={{ flex: 1, minWidth: 0, height: '100%' }}>
        <Paper withBorder radius="md" p="md" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          {composing ? (
            <Composer
              mode="new"
              contacts={contacts}
              sending={sending}
              onSend={doSend}
              onCancel={() => setComposing(false)}
            />
          ) : activeThread ? (
            <>
              <ThreadView thread={activeThread} onStar={onStar} />
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
      </Box>
    </Box>
  );
}
