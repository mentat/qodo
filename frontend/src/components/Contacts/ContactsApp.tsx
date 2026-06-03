import { useEffect, useMemo, useState } from 'react';
import {
  Grid, Stack, Group, Button, TextInput, ScrollArea, UnstyledButton, Text, Paper, Modal, Avatar,
} from '@mantine/core';
import { IconPlus, IconSearch, IconTrash, IconPencil } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { useContactStore } from '../../store/contactStore';
import type { Contact, ContactCreate } from '../../types/contact';
import { ContactForm } from './ContactForm';

export function ContactsApp() {
  const contacts = useContactStore((s) => s.contacts);
  const fetch = useContactStore((s) => s.fetch);
  const add = useContactStore((s) => s.add);
  const update = useContactStore((s) => s.update);
  const remove = useContactStore((s) => s.remove);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Contact | null>(null);
  const [q, setQ] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void fetch();
  }, [fetch]);

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return contacts;
    return contacts.filter((c) => `${c.name} ${c.company} ${c.email}`.toLowerCase().includes(s));
  }, [contacts, q]);

  const selected = contacts.find((c) => c.id === selectedId) ?? null;

  const submit = async (values: ContactCreate) => {
    setBusy(true);
    try {
      if (editing) await update(editing.id, values);
      else await add(values);
      setFormOpen(false);
      setEditing(null);
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    } finally {
      setBusy(false);
    }
  };

  const del = async (c: Contact) => {
    try {
      await remove(c.id);
      if (selectedId === c.id) setSelectedId(null);
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  return (
    <Grid gutter="md" style={{ height: 'calc(100vh - 92px)' }}>
      <Grid.Col span={{ base: 12, sm: 5 }} style={{ height: '100%' }}>
        <Stack gap="xs" style={{ height: '100%' }}>
          <Group gap="xs">
            <TextInput
              flex={1}
              placeholder="Search contacts..."
              leftSection={<IconSearch size={16} />}
              value={q}
              onChange={(e) => setQ(e.currentTarget.value)}
            />
            <Button
              leftSection={<IconPlus size={16} />}
              onClick={() => {
                setEditing(null);
                setFormOpen(true);
              }}
            >
              Add
            </Button>
          </Group>
          <ScrollArea style={{ flex: 1 }} type="auto">
            <Stack gap={4}>
              {filtered.map((c) => (
                <UnstyledButton
                  key={c.id}
                  onClick={() => setSelectedId(c.id)}
                  style={{
                    padding: 8,
                    borderRadius: 8,
                    background: c.id === selectedId ? 'var(--mantine-color-synthPurple-light)' : 'transparent',
                  }}
                >
                  <Group gap="sm" wrap="nowrap">
                    <Avatar color="synthPurple" radius="xl">
                      {c.name.charAt(0).toUpperCase()}
                    </Avatar>
                    <div>
                      <Text size="sm" fw={600}>
                        {c.name}
                      </Text>
                      <Text size="xs" c="dimmed">
                        {c.company || c.email}
                      </Text>
                    </div>
                  </Group>
                </UnstyledButton>
              ))}
            </Stack>
          </ScrollArea>
        </Stack>
      </Grid.Col>

      <Grid.Col span={{ base: 12, sm: 7 }} style={{ height: '100%' }}>
        <Paper withBorder radius="md" p="lg" style={{ height: '100%' }}>
          {selected ? (
            <Stack>
              <Group justify="space-between">
                <Group gap="sm">
                  <Avatar color="synthPurple" radius="xl" size="lg">
                    {selected.name.charAt(0).toUpperCase()}
                  </Avatar>
                  <div>
                    <Text fw={800} size="xl">
                      {selected.name}
                    </Text>
                    <Text c="dimmed">{selected.company}</Text>
                  </div>
                </Group>
                <Group gap={4}>
                  <Button
                    variant="light"
                    size="xs"
                    leftSection={<IconPencil size={14} />}
                    onClick={() => {
                      setEditing(selected);
                      setFormOpen(true);
                    }}
                  >
                    Edit
                  </Button>
                  <Button variant="light" color="red" size="xs" leftSection={<IconTrash size={14} />} onClick={() => del(selected)}>
                    Delete
                  </Button>
                </Group>
              </Group>
              {selected.email && <Text size="sm">✉️ {selected.email}</Text>}
              {selected.phone && <Text size="sm">📞 {selected.phone}</Text>}
              {selected.notes && (
                <Text size="sm" c="dimmed" style={{ whiteSpace: 'pre-wrap' }}>
                  {selected.notes}
                </Text>
              )}
            </Stack>
          ) : (
            <Group justify="center" align="center" style={{ height: '100%' }}>
              <Text c="dimmed">Select a contact.</Text>
            </Group>
          )}
        </Paper>
      </Grid.Col>

      <Modal opened={formOpen} onClose={() => setFormOpen(false)} title={editing ? 'Edit contact' : 'Add contact'}>
        <ContactForm contact={editing} onSubmit={submit} loading={busy} />
      </Modal>
    </Grid>
  );
}
