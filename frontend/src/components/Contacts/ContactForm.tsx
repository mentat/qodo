import { TextInput, Textarea, Button, Stack } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useEffect } from 'react';
import type { Contact, ContactCreate } from '../../types/contact';

interface Props {
  contact?: Contact | null;
  onSubmit: (values: ContactCreate) => void;
  loading?: boolean;
}

export function ContactForm({ contact, onSubmit, loading }: Props) {
  const form = useForm<ContactCreate>({
    initialValues: { name: '', email: '', phone: '', company: '', notes: '' },
    validate: { name: (v) => (v.trim().length === 0 ? 'Name is required' : null) },
  });

  useEffect(() => {
    if (contact) {
      form.setValues({
        name: contact.name,
        email: contact.email,
        phone: contact.phone,
        company: contact.company,
        notes: contact.notes,
      });
    } else {
      form.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [contact]);

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput label="Name" required {...form.getInputProps('name')} />
        <TextInput label="Email" {...form.getInputProps('email')} />
        <TextInput label="Phone" {...form.getInputProps('phone')} />
        <TextInput label="Company" {...form.getInputProps('company')} />
        <Textarea label="Notes" autosize minRows={2} {...form.getInputProps('notes')} />
        <Button type="submit" loading={loading}>
          {contact ? 'Update' : 'Add'} contact
        </Button>
      </Stack>
    </form>
  );
}
