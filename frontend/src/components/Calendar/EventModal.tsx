import { useEffect, useState } from 'react';
import { Modal, Stack, TextInput, Textarea, Switch, Group, Button } from '@mantine/core';
import { DateTimePicker } from '@mantine/dates';
import { notifications } from '@mantine/notifications';
import dayjs from 'dayjs';
import { useEventStore } from '../../store/eventStore';
import type { CalendarEvent } from '../../types/event';

interface Props {
  opened: boolean;
  onClose: () => void;
  event: CalendarEvent | null;
  slot: { start: Date; end: Date } | null;
}

// Mantine v8 date inputs use string values ('YYYY-MM-DD HH:mm:ss'); we keep
// that as-is in state and normalize to ISO 8601 (RFC3339) for the API.
const PICKER_FMT = 'YYYY-MM-DD HH:mm:ss';
const toPicker = (d: Date) => dayjs(d).format(PICKER_FMT);

export function EventModal({ opened, onClose, event, slot }: Props) {
  const create = useEventStore((s) => s.create);
  const update = useEventStore((s) => s.update);
  const remove = useEventStore((s) => s.remove);

  const [title, setTitle] = useState('');
  const [location, setLocation] = useState('');
  const [description, setDescription] = useState('');
  const [start, setStart] = useState<string | null>(null);
  const [end, setEnd] = useState<string | null>(null);
  const [allDay, setAllDay] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!opened) return;
    if (event) {
      setTitle(event.title);
      setLocation(event.location);
      setDescription(event.description);
      setStart(toPicker(event.start));
      setEnd(toPicker(event.end));
      setAllDay(event.allDay);
    } else {
      const s = slot?.start ?? new Date();
      const e = slot?.end ?? new Date(s.getTime() + 60 * 60 * 1000);
      setTitle('');
      setLocation('');
      setDescription('');
      setStart(toPicker(s));
      setEnd(toPicker(e));
      setAllDay(false);
    }
  }, [opened, event, slot]);

  const submit = async () => {
    if (!title.trim() || !start) return;
    setBusy(true);
    try {
      const startISO = dayjs(start).toISOString();
      const endISO = end ? dayjs(end).toISOString() : dayjs(start).add(1, 'hour').toISOString();
      const input = { title, location, description, start: startISO, end: endISO, allDay };
      if (event) await update(event.id, input);
      else await create(input);
      onClose();
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    } finally {
      setBusy(false);
    }
  };

  const del = async () => {
    if (!event) return;
    setBusy(true);
    try {
      await remove(event.id);
      onClose();
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title={event ? 'Edit event' : 'New event'}>
      <Stack>
        <TextInput label="Title" required value={title} onChange={(e) => setTitle(e.currentTarget.value)} />
        <DateTimePicker label="Start" value={start} onChange={setStart} />
        <DateTimePicker label="End" value={end} onChange={setEnd} />
        <Switch label="All day" checked={allDay} onChange={(e) => setAllDay(e.currentTarget.checked)} />
        <TextInput label="Location" value={location} onChange={(e) => setLocation(e.currentTarget.value)} />
        <Textarea label="Description" autosize minRows={2} value={description} onChange={(e) => setDescription(e.currentTarget.value)} />
        <Group justify="space-between">
          {event ? (
            <Button color="red" variant="light" loading={busy} onClick={del}>
              Delete
            </Button>
          ) : (
            <span />
          )}
          <Button loading={busy} onClick={submit}>
            {event ? 'Save' : 'Create'}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
