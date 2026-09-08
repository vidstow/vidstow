import { act, render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, test } from 'vitest';
import Modal from '../src/lib/components/Modal.svelte';
import { modal } from '../src/lib/stores.js';

afterEach(() => modal.set(null));

test('confirmation actions can open a new dialog without it being closed', async () => {
  modal.set({ kind: 'confirm', title: 'Continue?', message: 'Review first.', actions: [{ label: 'Continue', action: () => modal.set({ kind: 'error', title: 'Cannot continue', message: 'Choose a writable folder.' }) }] });
  render(Modal);
  await userEvent.setup().click(screen.getByRole('button', { name: 'Continue' }));
  expect(await screen.findByRole('dialog', { name: 'Cannot continue' })).toHaveTextContent('Choose a writable folder.');
});

test('confirmation traps focus and restores it after Escape', async () => {
  const user = userEvent.setup();
  const trigger = document.createElement('button');
  document.body.append(trigger);
  trigger.focus();
  modal.set({ kind: 'confirm', title: 'Continue?', message: 'Review first.' });
  render(Modal);
  const close = screen.getByRole('button', { name: 'Close' });
  await waitFor(() => expect(close).toHaveFocus());
  await user.tab({ shift: true });
  expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
  await user.tab();
  expect(close).toHaveFocus();
  await user.keyboard('{Escape}');
  await waitFor(() => expect(trigger).toHaveFocus());
  trigger.remove();
});

test('an asynchronous action failure has a visible recovery dialog', async () => {
  let reject!: (reason: Error) => void;
  modal.set({ kind: 'confirm', title: 'Continue?', message: 'Review first.', actions: [{ label: 'Continue', action: () => new Promise<void>((_, fail) => { reject = fail; }) }] });
  render(Modal);
  await userEvent.setup().click(screen.getByRole('button', { name: 'Continue' }));
  await act(() => reject(new Error('Operation failed.')));
  expect(await screen.findByRole('dialog', { name: 'Action could not finish' })).toHaveTextContent('Operation failed.');
});
