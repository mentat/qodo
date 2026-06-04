import { useEffect, useState } from 'react';
import { Modal, Stack, SimpleGrid, TextInput, Textarea, Switch, Group, Button } from '@mantine/core';
import { DateTimePicker, DateInput } from '@mantine/dates';
import { IconCalendarEvent, IconTrash } from '@tabler/icons-react';
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

// Mantine date inputs use string values; we hold them as-is and normalize to
// ISO 8601 (RFC3339) for the API.
const DATETIME_FMT = 'YYYY-MM-DD HH:mm:ss';
const DATE_FMT = 'YYYY-MM-DD';
const toPicker = (d: Date, allDay: boolean) => dayjs(d).format(allDay ? DATE_FMT : DATETIME_FMT);

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

  /* eslint-disable react-hooks/set-state-in-effect -- Opening a modal intentionally resets the editable draft. */
  useEffect(() => {
    if (!opened) return;
    const ad = event?.allDay ?? false;
    const s = event?.start ?? slot?.start ?? new Date();
    const e = event?.end ?? slot?.end ?? new Date(s.getTime() + 60 * 60 * 1000);
    setTitle(event?.title ?? '');
    setLocation(event?.location ?? '');
    setDescription(event?.description ?? '');
    setAllDay(ad);
    setStart(toPicker(s, ad));
    setEnd(toPicker(e, ad));
  }, [opened, event, slot]);
  /* eslint-enable react-hooks/set-state-in-effect */

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

  const dateProps = {
    leftSection: <IconCalendarEvent size={16} />,
    clearable: false,
    popoverProps: { withinPortal: true },
  };

  return (
    <Modal opened={opened} onClose={onClose} title={event ? 'Edit event' : 'New event'} size="lg" radius="md" centered>
      <Stack gap="md">
        <TextInput
          label="Title"
          placeholder="What's happening?"
          required
          size="md"
          data-autofocus
          value={title}
          onChange={(e) => setTitle(e.currentTarget.value)}
        />

        <Switch
          label="All-day event"
          checked={allDay}
          onChange={(e) => setAllDay(e.currentTarget.checked)}
        />

        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          {allDay ? (
            <>
              <DateInput {...dateProps} label="Starts" valueFormat="MMM D, YYYY" value={start} onChange={setStart} />
              <DateInput {...dateProps} label="Ends" valueFormat="MMM D, YYYY" value={end} onChange={setEnd} />
            </>
          ) : (
            <>
              <DateTimePicker
                {...dateProps}
                label="Starts"
                valueFormat="MMM D, YYYY · h:mm A"
                value={start}
                onChange={setStart}
                timePickerProps={{ format: '12h', withDropdown: true }}
              />
              <DateTimePicker
                {...dateProps}
                label="Ends"
                valueFormat="MMM D, YYYY · h:mm A"
                value={end}
                onChange={setEnd}
                timePickerProps={{ format: '12h', withDropdown: true }}
              />
            </>
          )}
        </SimpleGrid>

        <TextInput label="Location" placeholder="Where?" value={location} onChange={(e) => setLocation(e.currentTarget.value)} />
        <Textarea label="Description" placeholder="Notes, agenda, vibes…" autosize minRows={2} maxRows={6} value={description} onChange={(e) => setDescription(e.currentTarget.value)} />

        <Group justify="space-between" mt="xs">
          {event ? (
            <Button color="red" variant="light" leftSection={<IconTrash size={16} />} loading={busy} onClick={del}>
              Delete
            </Button>
          ) : (
            <span />
          )}
          <Group gap="xs">
            <Button variant="default" onClick={onClose}>
              Cancel
            </Button>
            <Button loading={busy} onClick={submit}>
              {event ? 'Save changes' : 'Create event'}
            </Button>
          </Group>
        </Group>
      </Stack>
    </Modal>
  );
}
