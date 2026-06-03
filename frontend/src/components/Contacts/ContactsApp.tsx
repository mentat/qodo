import { useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  Box, Stack, Group, Button, TextInput, ScrollArea, UnstyledButton, Text, Paper, Modal,
  Avatar, Center, ActionIcon, ThemeIcon, Divider,
} from '@mantine/core';
import { IconPlus, IconSearch, IconTrash, IconPencil, IconMail, IconPhone, IconBuilding, IconAddressBook } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { useContactStore } from '../../store/contactStore';
import type { Contact, ContactCreate } from '../../types/contact';
import { ContactForm } from './ContactForm';

function initials(name: string) {
  return name.trim().split(/\s+/).slice(0, 2).map((p) => p[0]?.toUpperCase() ?? '').join('') || '?';
}

function InfoRow({ icon, value }: { icon: ReactNode; value: string }) {
  return (
    <Group gap="sm" wrap="nowrap">
      <ThemeIcon variant="light" color="synthPurple" radius="xl" size="md">
        {icon}
      </ThemeIcon>
      <Text size="sm" style={{ wordBreak: 'break-word' }}>
        {value}
      </Text>
    </Group>
  );
}

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
    <Box style={{ display: 'flex', gap: 'var(--mantine-spacing-md)', height: 'calc(100vh - 92px)' }}>
      <Box style={{ flex: '0 0 300px', minWidth: 0, display: 'flex', flexDirection: 'column' }}>
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
        <ScrollArea style={{ flex: 1 }} type="auto" mt="xs">
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
                    {initials(c.name)}
                  </Avatar>
                  <div style={{ minWidth: 0 }}>
                    <Text size="sm" fw={600} truncate>
                      {c.name}
                    </Text>
                    <Text size="xs" c="dimmed" truncate>
                      {c.company || c.email}
                    </Text>
                  </div>
                </Group>
              </UnstyledButton>
            ))}
          </Stack>
        </ScrollArea>
      </Box>

      <Box style={{ flex: 1, minWidth: 0 }}>
        <Paper
          withBorder
          radius="md"
          style={{ height: '100%', minHeight: 460, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        >
          {selected ? (
            <>
              <Box
                p="xl"
                style={{
                  background: 'linear-gradient(135deg, var(--mantine-color-synthPurple-6), var(--mantine-color-neonPink-6))',
                  color: '#fff',
                }}
              >
                <Group justify="space-between" align="flex-start" wrap="nowrap">
                  <Group gap="md" wrap="nowrap">
                    <Avatar
                      size={72}
                      radius="xl"
                      styles={{ root: { background: 'rgba(255,255,255,0.2)', color: '#fff', border: '2px solid rgba(255,255,255,0.5)' }, placeholder: { color: '#fff', fontSize: 26, fontWeight: 800 } }}
                    >
                      {initials(selected.name)}
                    </Avatar>
                    <div style={{ minWidth: 0 }}>
                      <Text fw={800} fz={24} lineClamp={1}>
                        {selected.name}
                      </Text>
                      <Text fz="sm" style={{ opacity: 0.9 }} lineClamp={1}>
                        {selected.company || '—'}
                      </Text>
                    </div>
                  </Group>
                  <Group gap={6} wrap="nowrap">
                    <ActionIcon
                      variant="white"
                      aria-label="Edit"
                      onClick={() => {
                        setEditing(selected);
                        setFormOpen(true);
                      }}
                    >
                      <IconPencil size={16} />
                    </ActionIcon>
                    <ActionIcon variant="white" color="red" aria-label="Delete" onClick={() => del(selected)}>
                      <IconTrash size={16} />
                    </ActionIcon>
                  </Group>
                </Group>
              </Box>

              <Stack p="xl" gap="md" style={{ flex: 1 }}>
                {selected.email && <InfoRow icon={<IconMail size={16} />} value={selected.email} />}
                {selected.phone && <InfoRow icon={<IconPhone size={16} />} value={selected.phone} />}
                {selected.company && <InfoRow icon={<IconBuilding size={16} />} value={selected.company} />}
                {selected.notes && (
                  <>
                    <Divider label="Notes" labelPosition="left" />
                    <Text size="sm" c="dimmed" style={{ whiteSpace: 'pre-wrap' }}>
                      {selected.notes}
                    </Text>
                  </>
                )}
              </Stack>
            </>
          ) : (
            <Center style={{ flex: 1 }}>
              <Stack align="center" gap={6}>
                <IconAddressBook size={48} opacity={0.4} />
                <Text c="dimmed" size="sm">
                  Select a contact to see their details.
                </Text>
              </Stack>
            </Center>
          )}
        </Paper>
      </Box>

      <Modal opened={formOpen} onClose={() => setFormOpen(false)} title={editing ? 'Edit contact' : 'Add contact'}>
        <ContactForm contact={editing} onSubmit={submit} loading={busy} />
      </Modal>
    </Box>
  );
}
