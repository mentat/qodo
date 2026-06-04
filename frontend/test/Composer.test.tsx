import { describe, it, expect, mock } from 'bun:test';
import userEvent from '@testing-library/user-event';
import { screen } from '@testing-library/react';
import { renderWithMantine } from './render';
import { Composer } from '../src/components/Mail/Composer';
import type { SendInput } from '../src/api/mail';
import type { Contact } from '../src/types/contact';

function contact(overrides: Partial<Contact>): Contact {
  return {
    id: 'contact-1',
    userId: 'user-1',
    name: 'Capt. Nimbus',
    email: 'nimbus@synthwave.os',
    phone: '',
    company: 'Cloud Command',
    notes: '',
    characterId: 'captain-nimbus',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('Composer (new mode)', () => {
  it('fires onSend with the typed recipient, subject, and body', async () => {
    const calls: SendInput[] = [];
    const onSend = mock((input: SendInput) => {
      calls.push(input);
    });
    renderWithMantine(<Composer mode="new" sending={false} onSend={onSend} />);

    const user = userEvent.setup();
    await user.type(screen.getByPlaceholderText(/^To/), 'nimbus@synthwave.os');
    await user.type(screen.getByPlaceholderText('Subject'), 'Ahoy');
    await user.type(screen.getByPlaceholderText(/Write your message/), 'Hello captain');
    await user.click(screen.getByRole('button', { name: /send/i }));

    expect(onSend).toHaveBeenCalledTimes(1);
    expect(calls[0].to).toBe('nimbus@synthwave.os');
    expect(calls[0].subject).toBe('Ahoy');
    expect(calls[0].body).toBe('Hello captain');
  });

  it('sends the selected contact address from the recipient dropdown', async () => {
    const calls: SendInput[] = [];
    const onSend = mock((input: SendInput) => {
      calls.push(input);
    });
    renderWithMantine(
      <Composer
        mode="new"
        contacts={[contact({})]}
        sending={false}
        onSend={onSend}
      />,
    );

    const user = userEvent.setup();
    await user.type(screen.getByPlaceholderText(/^To/), 'Nimbus');
    await user.keyboard('{ArrowDown}{Enter}');
    await user.type(screen.getByPlaceholderText('Subject'), 'Ahoy');
    await user.type(screen.getByPlaceholderText(/Write your message/), 'Hello captain');
    await user.click(screen.getByRole('button', { name: /send/i }));

    expect(onSend).toHaveBeenCalledTimes(1);
    expect(calls[0].to).toBe('nimbus@synthwave.os');
    expect(calls[0].toName).toBe('Capt. Nimbus');
    expect(calls[0].characterId).toBe('captain-nimbus');
  });

  it('disables Send until a body is entered', async () => {
    const onSend = mock(() => {});
    renderWithMantine(<Composer mode="new" sending={false} onSend={onSend} />);
    const sendBtn = screen.getByRole('button', { name: /send/i });
    expect(sendBtn).toBeDisabled();
  });
});
