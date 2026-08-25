import { render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';

import SettingsPage from '../src/pages/Settings.svelte';
import { ffmpeg, settings } from '../src/lib/stores.js';

function baseSettings() {
  return {
    downloadFolder: '/tmp/downloads', ffmpegPath: '', windowWidth: 1180, windowHeight: 760,
    downloadConcurrency: 2, perVideoSubfolder: true, confirmBeforeDownload: false,
    outputOptions: {
      subtitleMode: '' as const, subtitleSidecar: false, subtitleAutoCaptions: true,
      embedThumbnail: true, embedChapters: true,
    },
    automaticDiagnostics: 'disabled' as const,
  };
}

describe('Settings subtitle defaults', () => {
  beforeEach(() => {
    settings.set(baseSettings());
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    const UpdateSettings = vi.fn(async (next: any) => next);
    (window as any).go = { main: { App: { UpdateSettings } } };
  });

  test('fresh Settings is Off and artwork and chapters are automatic, not toggles', () => {
    render(SettingsPage);
    expect(screen.getByRole('heading', { name: 'Video files' })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Off' })).toBeChecked();
    expect(screen.getByRole('radio', { name: 'On' })).not.toBeChecked();
    expect(screen.getByText(/Thumbnail artwork and chapter markers are included automatically/)).toBeInTheDocument();
    expect(screen.queryByLabelText(/Embed thumbnail artwork/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/Embed chapter markers/i)).not.toBeInTheDocument();
    expect(screen.queryByText('Confirm before starting downloads')).not.toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: /Also save an \.srt file/ })).not.toBeChecked();
  });

  test('legacy subtitle-file defaults normalize to embedded plus SRT when toggled Off and On', async () => {
    const user = userEvent.setup();
    settings.set({
      ...baseSettings(),
      outputOptions: {
        subtitleMode: 'sidecar', subtitleSidecar: false, subtitleAutoCaptions: false,
        subtitleFormat: 'vtt', subtitleLanguages: ['en'], embedThumbnail: true, embedChapters: true,
      },
    });
    render(SettingsPage);
    const UpdateSettings = (window as any).go.main.App.UpdateSettings;

    expect(screen.getByRole('radio', { name: 'On' })).toBeChecked();
    expect(screen.getByRole('checkbox', { name: /Also save an \.srt file/ })).toBeChecked();
    await user.click(screen.getByRole('radio', { name: 'Off' }));
    await waitFor(() => expect(UpdateSettings).toHaveBeenCalledTimes(1));
    expect(UpdateSettings.mock.calls[0][0].outputOptions).toMatchObject({
      subtitleMode: '', subtitleSidecar: true, subtitleFormat: 'srt', subtitleLanguages: ['en'],
    });

    await user.click(screen.getByRole('radio', { name: 'On' }));
    await waitFor(() => expect(UpdateSettings).toHaveBeenCalledTimes(2));
    expect(UpdateSettings.mock.calls[1][0].outputOptions).toMatchObject({
      subtitleMode: 'embed', subtitleSidecar: true, subtitleAutoCaptions: true,
      subtitleFormat: 'srt', subtitleLanguages: ['en'], embedThumbnail: true, embedChapters: true,
    });
  });
});
