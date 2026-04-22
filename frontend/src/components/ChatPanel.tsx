import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import {
  ActionIcon,
  Drawer,
  ScrollArea,
  Stack,
  Textarea,
  Box,
  Text,
  Group,
  Button,
  Tooltip,
} from '@mantine/core';
import { IconSend, IconTrash } from '@tabler/icons-react';
import { useChatStore } from '../store/chatStore';
import type { ChatMessage } from '../api/agent';
import { RobotAscii } from './RobotAscii';

interface Props {
  opened: boolean;
  onClose: () => void;
}

export function ChatPanel({ opened, onClose }: Props) {
  const messages = useChatStore((s) => s.messages);
  const sending = useChatStore((s) => s.sending);
  const loading = useChatStore((s) => s.loading);
  const error = useChatStore((s) => s.error);
  const load = useChatStore((s) => s.load);
  const send = useChatStore((s) => s.send);
  const reset = useChatStore((s) => s.reset);
  const [text, setText] = useState('');
  const viewportRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (opened) load();
  }, [opened, load]);

  // Scroll to the bottom when the transcript grows. useLayoutEffect runs
  // synchronously after the DOM has been updated so scrollHeight already
  // reflects the new messages; rAF gives Mantine's ScrollArea one frame
  // to finalize its inner layout.
  useLayoutEffect(() => {
    const el = viewportRef.current;
    if (!el) return;
    const scroll = () => el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
    scroll();
    const raf = requestAnimationFrame(scroll);
    return () => cancelAnimationFrame(raf);
  }, [messages, sending]);

  const handleSend = () => {
    if (!text.trim() || sending) return;
    void send(text);
    setText('');
  };

  return (
    <Drawer
      opened={opened}
      onClose={onClose}
      position="right"
      size={400}
      title={<Text fw={700} c="teal">Marvin AI</Text>}
      overlayProps={{ backgroundOpacity: 0.2, blur: 2 }}
      styles={{ body: { display: 'flex', flexDirection: 'column', height: 'calc(100vh - 60px)', padding: 12, gap: 10 } }}
    >
      <RobotAscii thinking={sending} />
      <Group justify="space-between" gap={4}>
        <Text size="xs" c="dimmed">
          {sending ? '*whirrrr* processing…' : loading ? 'booting…' : 'online (barely)'}
        </Text>
        <Tooltip label="Wipe chat history">
          <ActionIcon variant="subtle" color="red" onClick={() => void reset()} size="sm">
            <IconTrash size={14} />
          </ActionIcon>
        </Tooltip>
      </Group>

      <ScrollArea viewportRef={viewportRef} style={{ flex: 1, minHeight: 0 }} type="auto">
        <Stack gap={6} pb="md">
          {messages.length === 0 && !loading && (
            <Text size="sm" c="dimmed" ta="center" py="md">
              BEEP. Say hi to Marvin — ask about news, Wikipedia, or your todos.
            </Text>
          )}
          {messages.map((m) => (
            <Bubble key={m.id} message={m} />
          ))}
          {sending && (
            <Bubble
              message={{
                id: 'typing',
                userId: 'marvin',
                role: 'assistant',
                content: '▯▯▯',
                createdAt: new Date().toISOString(),
              }}
              pending
            />
          )}
        </Stack>
      </ScrollArea>

      {error && (
        <Text size="xs" c="red">
          {error}
        </Text>
      )}

      <Group gap={6} align="flex-end">
        <Textarea
          value={text}
          onChange={(e) => setText(e.currentTarget.value)}
          placeholder="Tell Marvin what to do…"
          autosize
          minRows={1}
          maxRows={5}
          style={{ flex: 1 }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              handleSend();
            }
          }}
          disabled={sending}
        />
        <Button
          onClick={handleSend}
          loading={sending}
          disabled={sending || !text.trim()}
          leftSection={<IconSend size={14} />}
          size="sm"
        >
          Send
        </Button>
      </Group>
    </Drawer>
  );
}

function Bubble({ message, pending }: { message: ChatMessage; pending?: boolean }) {
  const isUser = message.role === 'user';
  const isScreened = !!message.screened;
  return (
    <Box
      style={{
        alignSelf: isUser ? 'flex-end' : 'flex-start',
        maxWidth: '85%',
        background: isUser ? 'var(--mantine-color-blue-light)' : '#0a1512',
        color: isUser ? undefined : '#9effc7',
        border: isScreened ? '1px dashed #ff7b7b' : 'none',
        borderRadius: 10,
        padding: '8px 12px',
        fontFamily: isUser ? undefined : 'ui-monospace, SFMono-Regular, Menlo, monospace',
        fontSize: 13,
        whiteSpace: 'pre-wrap',
        opacity: pending ? 0.6 : 1,
      }}
    >
      {message.content}
    </Box>
  );
}
