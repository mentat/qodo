import { Card, Group, Checkbox, Text, Badge, ActionIcon, Stack } from '@mantine/core';
import { IconGripVertical, IconPencil, IconTrash } from '@tabler/icons-react';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import type { Todo } from '../types/todo';
import type { DraggableProvided } from '@hello-pangea/dnd';

dayjs.extend(relativeTime);

const priorityColors: Record<string, string> = {
  high: 'neonPink',
  medium: 'hotYellow',
  low: 'electricBlue',
};

interface TodoItemProps {
  todo: Todo;
  provided: DraggableProvided;
  onToggle: (id: string, completed: boolean) => void;
  onEdit: (todo: Todo) => void;
  onDelete: (id: string) => void;
}

export function TodoItem({ todo, provided, onToggle, onEdit, onDelete }: TodoItemProps) {
  const isOverdue = todo.dueDate && !todo.completed && dayjs(todo.dueDate).isBefore(dayjs());

  return (
    <Card
      ref={provided.innerRef}
      {...provided.draggableProps}
      withBorder
      shadow="xs"
      mb="xs"
      padding="sm"
      opacity={todo.completed ? 0.6 : 1}
    >
      <Group justify="space-between" wrap="nowrap">
        <Group wrap="nowrap" gap="sm">
          <ActionIcon
            variant="subtle"
            color="gray"
            size="sm"
            style={{ cursor: 'grab' }}
            {...provided.dragHandleProps}
          >
            <IconGripVertical size={16} />
          </ActionIcon>

          <Checkbox
            checked={todo.completed}
            onChange={() => onToggle(todo.id, !todo.completed)}
            size="md"
          />

          <Stack gap={2}>
            <Text
              size="sm"
              fw={500}
              td={todo.completed ? 'line-through' : undefined}
              c={todo.completed ? 'dimmed' : undefined}
            >
              {todo.title}
            </Text>
            {todo.description && (
              <Text size="xs" c="dimmed" lineClamp={1}>
                {todo.description}
              </Text>
            )}
            <Group gap="xs">
              <Badge size="xs" color={priorityColors[todo.priority]} variant="light">
                {todo.priority}
              </Badge>
              {todo.category && (
                <Badge size="xs" color="teal" variant="light">
                  {todo.category}
                </Badge>
              )}
              {todo.dueDate && (
                <Text size="xs" c={isOverdue ? 'red' : 'dimmed'}>
                  {dayjs(todo.dueDate).fromNow()}
                </Text>
              )}
            </Group>
          </Stack>
        </Group>

        <Group gap="xs" wrap="nowrap">
          <ActionIcon variant="subtle" color="blue" size="sm" onClick={() => onEdit(todo)}>
            <IconPencil size={16} />
          </ActionIcon>
          <ActionIcon variant="subtle" color="red" size="sm" onClick={() => onDelete(todo.id)}>
            <IconTrash size={16} />
          </ActionIcon>
        </Group>
      </Group>
    </Card>
  );
}
