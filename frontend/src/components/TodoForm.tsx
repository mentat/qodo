import { TextInput, Textarea, Select, Button, Stack } from '@mantine/core';
import { DatePickerInput } from '@mantine/dates';
import { useForm } from '@mantine/form';
import { useEffect } from 'react';
import type { Todo } from '../types/todo';

interface TodoFormProps {
  todo?: Todo | null;
  onSubmit: (values: FormValues) => void;
  loading?: boolean;
}

export interface FormValues {
  title: string;
  description: string;
  priority: 'low' | 'medium' | 'high';
  category: string;
  dueDate: Date | null;
}

export function TodoForm({ todo, onSubmit, loading }: TodoFormProps) {
  const form = useForm<FormValues>({
    initialValues: {
      title: '',
      description: '',
      priority: 'medium',
      category: '',
      dueDate: null,
    },
    validate: {
      title: (v) => (v.trim().length === 0 ? 'Title is required' : null),
    },
  });

  useEffect(() => {
    if (todo) {
      form.setValues({
        title: todo.title,
        description: todo.description,
        priority: todo.priority,
        category: todo.category,
        dueDate: todo.dueDate ? new Date(todo.dueDate) : null,
      });
    } else {
      form.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [todo]);

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput label="Title" placeholder="What needs to be done?" required {...form.getInputProps('title')} />
        <Textarea label="Description" placeholder="Optional details..." autosize minRows={2} {...form.getInputProps('description')} />
        <Select
          label="Priority"
          data={[
            { value: 'low', label: 'Low' },
            { value: 'medium', label: 'Medium' },
            { value: 'high', label: 'High' },
          ]}
          {...form.getInputProps('priority')}
        />
        <TextInput label="Category" placeholder="e.g. work, personal" {...form.getInputProps('category')} />
        <DatePickerInput label="Due date" placeholder="Pick a date" clearable {...form.getInputProps('dueDate')} />
        <Button type="submit" loading={loading}>
          {todo ? 'Update' : 'Create'} Todo
        </Button>
      </Stack>
    </form>
  );
}
