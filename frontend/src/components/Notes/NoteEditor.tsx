import { useEffect, useState } from 'react';
import { Stack, Group, TextInput, Textarea, SegmentedControl, Button, Paper, Box } from '@mantine/core';
import { IconDeviceFloppy, IconTrash } from '@tabler/icons-react';
import Markdown from 'react-markdown';
import type { Note } from '../../types/note';

interface Props {
  note: Note | null;
  onSave: (title: string, body: string) => Promise<void>;
  onDelete?: () => void;
}

export function NoteEditor({ note, onSave, onDelete }: Props) {
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [tab, setTab] = useState<'write' | 'preview'>('write');
  const [busy, setBusy] = useState(false);

  /* eslint-disable react-hooks/set-state-in-effect -- Switching notes intentionally resets this local editor draft. */
  useEffect(() => {
    setTitle(note?.title ?? '');
    setBody(note?.body ?? '');
    setTab('write');
  }, [note]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const save = async () => {
    if (!title.trim() && !body.trim()) return;
    setBusy(true);
    try {
      await onSave(title, body);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Stack gap="xs" style={{ flex: 1, minHeight: 0 }}>
      <Group justify="space-between">
        <TextInput
          flex={1}
          placeholder="Note title"
          value={title}
          onChange={(e) => setTitle(e.currentTarget.value)}
          variant="unstyled"
          styles={{ input: { fontSize: 20, fontWeight: 800 } }}
        />
        <SegmentedControl
          size="xs"
          value={tab}
          onChange={(v) => setTab(v as 'write' | 'preview')}
          data={[
            { label: 'Write', value: 'write' },
            { label: 'Preview', value: 'preview' },
          ]}
        />
      </Group>

      {tab === 'write' ? (
        <Textarea
          flex={1}
          placeholder="Write in markdown…"
          value={body}
          onChange={(e) => setBody(e.currentTarget.value)}
          styles={{ wrapper: { height: '100%' }, input: { height: '100%', fontFamily: 'ui-monospace, monospace' } }}
        />
      ) : (
        <Paper withBorder radius="sm" p="md" style={{ flex: 1, overflow: 'auto' }}>
          <Box className="md-preview">
            <Markdown>{body || '_Nothing to preview yet._'}</Markdown>
          </Box>
        </Paper>
      )}

      <Group justify="space-between">
        {onDelete ? (
          <Button variant="light" color="red" leftSection={<IconTrash size={14} />} onClick={onDelete}>
            Delete
          </Button>
        ) : (
          <span />
        )}
        <Button leftSection={<IconDeviceFloppy size={16} />} loading={busy} onClick={save}>
          Save
        </Button>
      </Group>
    </Stack>
  );
}
