import { useEffect, useState } from 'react';
import { Grid, Stack, Button, ScrollArea, UnstyledButton, Text, Paper, Group } from '@mantine/core';
import { IconPlus } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import dayjs from 'dayjs';
import { useNoteStore } from '../../store/noteStore';
import type { Note } from '../../types/note';
import { NoteEditor } from './NoteEditor';

export function NotesApp() {
  const notes = useNoteStore((s) => s.notes);
  const fetch = useNoteStore((s) => s.fetch);
  const add = useNoteStore((s) => s.add);
  const update = useNoteStore((s) => s.update);
  const remove = useNoteStore((s) => s.remove);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    void fetch();
  }, [fetch]);

  const selected: Note | null = creating ? null : notes.find((n) => n.id === selectedId) ?? null;

  const handleSave = async (title: string, body: string) => {
    try {
      if (creating || !selected) {
        const n = await add({ title, body });
        setCreating(false);
        setSelectedId(n.id);
      } else {
        await update(selected.id, { title, body });
      }
      notifications.show({ title: 'Saved', message: 'Note saved', color: 'neonGreen' });
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  const handleDelete = async () => {
    if (!selected) return;
    try {
      await remove(selected.id);
      setSelectedId(null);
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  const showEditor = creating || selected;

  return (
    <Grid gutter="md" style={{ height: 'calc(100vh - 92px)' }}>
      <Grid.Col span={{ base: 12, sm: 4 }} style={{ height: '100%' }}>
        <Stack gap="xs" style={{ height: '100%' }}>
          <Button
            leftSection={<IconPlus size={16} />}
            variant="light"
            onClick={() => {
              setCreating(true);
              setSelectedId(null);
            }}
          >
            New note
          </Button>
          <ScrollArea style={{ flex: 1 }} type="auto">
            <Stack gap={4}>
              {notes.map((n) => (
                <UnstyledButton
                  key={n.id}
                  onClick={() => {
                    setCreating(false);
                    setSelectedId(n.id);
                  }}
                  style={{
                    padding: 8,
                    borderRadius: 8,
                    background: !creating && n.id === selectedId ? 'var(--mantine-color-synthPurple-light)' : 'transparent',
                  }}
                >
                  <Text size="sm" fw={600} truncate>
                    {n.title || 'Untitled'}
                  </Text>
                  <Text size="11px" c="dimmed">
                    {dayjs(n.updatedAt).format('MMM D, h:mm A')}
                  </Text>
                </UnstyledButton>
              ))}
            </Stack>
          </ScrollArea>
        </Stack>
      </Grid.Col>

      <Grid.Col span={{ base: 12, sm: 8 }} style={{ height: '100%' }}>
        <Paper withBorder radius="md" p="md" style={{ height: '100%' }}>
          {showEditor ? (
            <NoteEditor
              note={selected}
              onSave={handleSave}
              onDelete={selected ? handleDelete : undefined}
            />
          ) : (
            <Group justify="center" align="center" style={{ height: '100%' }}>
              <Text c="dimmed">Select a note or create a new one.</Text>
            </Group>
          )}
        </Paper>
      </Grid.Col>
    </Grid>
  );
}
