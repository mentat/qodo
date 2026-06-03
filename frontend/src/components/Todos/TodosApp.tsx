import { useState, useMemo, useEffect } from 'react';
import { Button, Modal, Group, Container, TextInput } from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { IconPlus, IconSearch } from '@tabler/icons-react';
import { TodoFilters } from '../TodoFilters';
import { TodoList } from '../TodoList';
import { TodoForm, type FormValues } from '../TodoForm';
import { useTodoStore, filterTodos, deriveCategories } from '../../store/todoStore';
import type { Todo } from '../../types/todo';

export function TodosApp() {
  const [formOpen, setFormOpen] = useState(false);
  const [editingTodo, setEditingTodo] = useState<Todo | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [search, setSearch] = useState('');
  const [debounced] = useDebouncedValue(search, 300);

  const todos = useTodoStore((s) => s.todos);
  const searchHits = useTodoStore((s) => s.searchHits);
  const loading = useTodoStore((s) => s.loading);
  const priorityFilter = useTodoStore((s) => s.priorityFilter);
  const categoryFilter = useTodoStore((s) => s.categoryFilter);
  const statusFilter = useTodoStore((s) => s.statusFilter);
  const addTodo = useTodoStore((s) => s.addTodo);
  const updateTodo = useTodoStore((s) => s.updateTodo);
  const toggleTodo = useTodoStore((s) => s.toggleTodo);
  const removeTodo = useTodoStore((s) => s.removeTodo);
  const reorderTodos = useTodoStore((s) => s.reorderTodos);
  const setSearchQuery = useTodoStore((s) => s.setSearchQuery);
  const setPriorityFilter = useTodoStore((s) => s.setPriorityFilter);
  const setCategoryFilter = useTodoStore((s) => s.setCategoryFilter);
  const setStatusFilter = useTodoStore((s) => s.setStatusFilter);

  useEffect(() => {
    setSearchQuery(debounced);
  }, [debounced, setSearchQuery]);

  const filteredTodos = useMemo(() => {
    if (searchHits !== null) {
      if (!categoryFilter) return searchHits;
      return searchHits.filter((t) => t.category === categoryFilter);
    }
    return filterTodos(todos, '', priorityFilter, categoryFilter, statusFilter);
  }, [todos, searchHits, priorityFilter, categoryFilter, statusFilter]);

  const categories = useMemo(() => deriveCategories(todos), [todos]);

  const handleSubmit = async (values: FormValues) => {
    setSubmitting(true);
    try {
      const payload = { ...values, dueDate: values.dueDate ? values.dueDate.toISOString() : null };
      if (editingTodo) {
        await updateTodo(editingTodo.id, payload);
        notifications.show({ title: 'Updated', message: 'Todo updated', color: 'neonGreen' });
      } else {
        await addTodo(payload);
        notifications.show({ title: 'Created', message: 'Todo created', color: 'neonGreen' });
      }
      setFormOpen(false);
      setEditingTodo(null);
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggle = async (id: string, completed: boolean) => {
    try {
      await toggleTodo(id, completed);
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await removeTodo(id);
      notifications.show({ title: 'Deleted', message: 'Todo deleted', color: 'orange' });
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  const handleEdit = (todo: Todo) => {
    setEditingTodo(todo);
    setFormOpen(true);
  };

  const handleReorder = async (startIndex: number, endIndex: number) => {
    try {
      await reorderTodos(startIndex, endIndex);
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  return (
    <Container size="md">
      <Group justify="space-between" mb="md" wrap="wrap">
        <Group gap="sm">
          <TextInput
            placeholder="Search todos..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
            w={220}
          />
          <TodoFilters
            priority={priorityFilter}
            category={categoryFilter}
            status={statusFilter}
            categories={categories}
            onPriorityChange={setPriorityFilter}
            onCategoryChange={setCategoryFilter}
            onStatusChange={setStatusFilter}
          />
        </Group>
        <Button
          leftSection={<IconPlus size={16} />}
          onClick={() => {
            setEditingTodo(null);
            setFormOpen(true);
          }}
        >
          Add Todo
        </Button>
      </Group>

      <TodoList
        todos={filteredTodos}
        loading={loading}
        onToggle={handleToggle}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onReorder={handleReorder}
      />

      <Modal
        opened={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditingTodo(null);
        }}
        title={editingTodo ? 'Edit Todo' : 'New Todo'}
      >
        <TodoForm todo={editingTodo} onSubmit={handleSubmit} loading={submitting} />
      </Modal>
    </Container>
  );
}
