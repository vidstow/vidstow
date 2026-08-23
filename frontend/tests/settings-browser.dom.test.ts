import { render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';

import Settings from '../src/pages/Settings.svelte';
import { ffmpeg, modal, settings } from '../src/lib/stores.js';

const source = { bindingRef: 'binding-chrome', browser: 'chrome', label: 'Chrome — Default', enabled: true, default: true };

function installBindings() {
  const GetBrowserSourceOptions = vi.fn(async () => [{ id: 'chrome-default', browser: 'chrome', label: 'Chrome — Default' }]);
  const ListBrowserSources = vi.fn(async () => [source]);
  const ConfigureBrowserSource = vi.fn(async () => source);
  const CheckBrowserSource = vi.fn(async () => ({ status: 'ready', label: 'Chrome — Default', message: 'Browser source is ready to be supplied.', ready: true, partial: false }));
  const PreviewForgetBrowserSource = vi.fn(async () => ({ bindingRef: source.bindingRef, label: source.label, jobs: 3, collections: 1, active: 1 }));
  const ForgetBrowserSource = vi.fn(async () => 3);
  (window as any).go = { main: { App: { GetBrowserSourceOptions, ListBrowserSources, ConfigureBrowserSource, CheckBrowserSource, PreviewForgetBrowserSource, ForgetBrowserSource } } };
  return { ConfigureBrowserSource, CheckBrowserSource, PreviewForgetBrowserSource, ForgetBrowserSource };
}

describe('Settings browser access', () => {
  beforeEach(() => {
    modal.set(null);
    settings.update((current) => ({ ...current, downloadFolder: '/tmp/downloads' }));
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7.0', ffprobePath: '/usr/bin/ffprobe', message: '' });
  });

  test('requires explicit consent and performs a bounded local source check', async () => {
    const user = userEvent.setup();
    const { ConfigureBrowserSource, CheckBrowserSource } = installBindings();
    render(Settings);

    const button = await screen.findByRole('button', { name: 'Configure and check' });
    expect(button).toBeDisabled();
    expect(screen.getByText(/does not persist cookie values/i)).toBeInTheDocument();
    expect(screen.getByText(/permission or credential-store prompt/i)).toBeInTheDocument();

    await user.click(screen.getByRole('checkbox', { name: /I authorize VidStow/ }));
    expect(button).toBeEnabled();
    await user.click(button);

    await waitFor(() => expect(ConfigureBrowserSource).toHaveBeenCalledWith('chrome-default', 1));
    expect(CheckBrowserSource).toHaveBeenCalledWith('binding-chrome');
    expect(await screen.findByText(/Browser source is ready to be supplied/)).toBeInTheDocument();
  });

  test('previews exact dependent job and collection counts before forgetting', async () => {
    const user = userEvent.setup();
    const { PreviewForgetBrowserSource } = installBindings();
    render(Settings);

    await user.click(await screen.findByRole('button', { name: 'Forget…' }));
    await waitFor(() => expect(PreviewForgetBrowserSource).toHaveBeenCalledWith('binding-chrome'));
    const confirmation = get(modal);
    expect(confirmation?.title).toBe('Pause downloads using Chrome — Default');
    expect(confirmation?.message).toContain('3 downloads in 1 collection');
    expect(confirmation?.message).toContain('1 active download must be paused first');
    expect(confirmation?.actions).toBeUndefined();
  });
});
