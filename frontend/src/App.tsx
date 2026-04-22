import { useState, useEffect, useMemo } from 'react';
import { AppShell, Button, Modal, Group, Container, Center, Loader } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconPlus } from '@tabler/icons-react';
import { useAuth } from './hooks/useAuth';
import { LoginPage } from './components/LoginPage';
import { Header } from './components/Header';
import { TodoFilters } from './components/TodoFilters';
import { TodoList } from './components/TodoList';
import { TodoForm, type FormValues } from './components/TodoForm';
import { ChatPanel } from './components/ChatPanel';
import { useTodoStore, filterTodos, deriveCategories } from './store/todoStore';
import type { Todo } from './types/todo';

export default function App() {
  const { user, loading: authLoading, signOut } = useAuth();
  const [formOpen, setFormOpen] = useState(false);
  const [chatOpen, setChatOpen] = useState(false);
  const [editingTodo, setEditingTodo] = useState<Todo | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const todos = useTodoStore((s) => s.todos);
  const searchHits = useTodoStore((s) => s.searchHits);
  const loading = useTodoStore((s) => s.loading);
  const priorityFilter = useTodoStore((s) => s.priorityFilter);
  const categoryFilter = useTodoStore((s) => s.categoryFilter);
  const statusFilter = useTodoStore((s) => s.statusFilter);
  const fetchTodos = useTodoStore((s) => s.fetchTodos);
  const addTodo = useTodoStore((s) => s.addTodo);
  const updateTodo = useTodoStore((s) => s.updateTodo);
  const toggleTodo = useTodoStore((s) => s.toggleTodo);
  const removeTodo = useTodoStore((s) => s.removeTodo);
  const reorderTodos = useTodoStore((s) => s.reorderTodos);
  const setSearchQuery = useTodoStore((s) => s.setSearchQuery);
  const setPriorityFilter = useTodoStore((s) => s.setPriorityFilter);
  const setCategoryFilter = useTodoStore((s) => s.setCategoryFilter);
  const setStatusFilter = useTodoStore((s) => s.setStatusFilter);

  // When a search is active, searchHits is the authoritative set — already
  // stemmed-matched + completed/priority-filtered by the backend. We still
  // apply category client-side since it's not part of the search index.
  const filteredTodos = useMemo(() => {
    if (searchHits !== null) {
      if (!categoryFilter) return searchHits;
      return searchHits.filter((t) => t.category === categoryFilter);
    }
    return filterTodos(todos, '', priorityFilter, categoryFilter, statusFilter);
  }, [todos, searchHits, priorityFilter, categoryFilter, statusFilter]);

  const categories = useMemo(() => deriveCategories(todos), [todos]);

  useEffect(() => {
    if (user) fetchTodos();
  }, [user, fetchTodos]);

  const handleSubmit = async (values: FormValues) => {
    setSubmitting(true);
    try {
      const payload = {
        ...values,
        dueDate: values.dueDate ? values.dueDate.toISOString() : null,
      };
      if (editingTodo) {
        await updateTodo(editingTodo.id, payload);
        notifications.show({ title: 'Updated', message: 'Todo updated', color: 'green' });
      } else {
        await addTodo(payload);
        notifications.show({ title: 'Created', message: 'Todo created', color: 'green' });
      }
      setFormOpen(false);
      setEditingTodo(null);
    } catch (e: any) {
      notifications.show({ title: 'Error', message: e.message, color: 'red' });
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggle = async (id: string, completed: boolean) => {
    try {
      await toggleTodo(id, completed);
    } catch (e: any) {
      notifications.show({ title: 'Error', message: e.message, color: 'red' });
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await removeTodo(id);
      notifications.show({ title: 'Deleted', message: 'Todo deleted', color: 'orange' });
    } catch (e: any) {
      notifications.show({ title: 'Error', message: e.message, color: 'red' });
    }
  };

  const handleEdit = (todo: Todo) => {
    setEditingTodo(todo);
    setFormOpen(true);
  };

  const handleReorder = async (startIndex: number, endIndex: number) => {
    try {
      await reorderTodos(startIndex, endIndex);
    } catch (e: any) {
      notifications.show({ title: 'Error', message: e.message, color: 'red' });
    }
  };

  if (authLoading) {
    return (
      <Center mih="100vh">
        <Loader size="lg" />
      </Center>
    );
  }
  if (!user) return <LoginPage />;

  return (
    <AppShell header={{ height: 60 }} padding="md">
      <AppShell.Header>
        <Header
          user={user}
          onSearch={setSearchQuery}
          onSignOut={signOut}
          onOpenChat={() => setChatOpen(true)}
        />
      </AppShell.Header>

      <AppShell.Main>
        <Container size="md">
          <Group justify="space-between" mb="md">
            <TodoFilters
              priority={priorityFilter}
              category={categoryFilter}
              status={statusFilter}
              categories={categories}
              onPriorityChange={setPriorityFilter}
              onCategoryChange={setCategoryFilter}
              onStatusChange={setStatusFilter}
            />
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
        </Container>

        <ChatPanel opened={chatOpen} onClose={() => setChatOpen(false)} />

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
      </AppShell.Main>
    </AppShell>
  );
}
