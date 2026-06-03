import { useMemo, useState } from 'react';
import { Calendar, dayjsLocalizer, Views } from 'react-big-calendar';
import dayjs from 'dayjs';
import { Box } from '@mantine/core';
import { useEventStore } from '../../store/eventStore';
import { useTodoStore } from '../../store/todoStore';
import type { CalendarEvent } from '../../types/event';
import { EventModal } from './EventModal';

const localizer = dayjsLocalizer(dayjs);

export function CalendarApp() {
  const events = useEventStore((s) => s.events);
  const todos = useTodoStore((s) => s.todos);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<CalendarEvent | null>(null);
  const [slot, setSlot] = useState<{ start: Date; end: Date } | null>(null);

  // Merge real events with due-dated todos (rendered read-only as all-day).
  const merged = useMemo<CalendarEvent[]>(() => {
    const todoItems: CalendarEvent[] = todos
      .filter((t) => t.dueDate)
      .map((t) => {
        const d = new Date(t.dueDate as string);
        return {
          id: `todo-${t.id}`,
          userId: t.userId,
          title: `📋 ${t.title}`,
          description: t.description,
          location: '',
          start: d,
          end: d,
          allDay: true,
          color: '#39FF14',
          kind: 'todo' as const,
        };
      });
    return [...events, ...todoItems];
  }, [events, todos]);

  return (
    <Box style={{ height: 'calc(100vh - 92px)' }} className="synthwave-calendar">
      <Calendar<CalendarEvent>
        localizer={localizer}
        events={merged}
        startAccessor="start"
        endAccessor="end"
        titleAccessor="title"
        views={[Views.MONTH, Views.WEEK, Views.DAY]}
        defaultView={Views.MONTH}
        popup
        selectable
        style={{ height: '100%' }}
        eventPropGetter={(event) => ({
          style: {
            backgroundColor: event.color,
            border: 'none',
            color: '#0a0a12',
            fontWeight: 600,
          },
        })}
        onSelectEvent={(event) => {
          if (event.kind === 'todo') return;
          setEditing(event);
          setSlot(null);
          setModalOpen(true);
        }}
        onSelectSlot={(info) => {
          setEditing(null);
          setSlot({ start: info.start, end: info.end });
          setModalOpen(true);
        }}
      />
      <EventModal opened={modalOpen} onClose={() => setModalOpen(false)} event={editing} slot={slot} />
    </Box>
  );
}
