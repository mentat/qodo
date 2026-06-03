import { describe, it, expect, mock } from 'bun:test';
import userEvent from '@testing-library/user-event';
import { screen } from '@testing-library/react';
import { renderWithMantine } from './render';
import { Composer } from '../src/components/Mail/Composer';
import type { SendInput } from '../src/api/mail';

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

  it('disables Send until a body is entered', async () => {
    const onSend = mock((_: SendInput) => {});
    renderWithMantine(<Composer mode="new" sending={false} onSend={onSend} />);
    const sendBtn = screen.getByRole('button', { name: /send/i });
    expect(sendBtn).toBeDisabled();
  });
});
