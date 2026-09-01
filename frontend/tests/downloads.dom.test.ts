import { render, screen } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import Banner from '../src/lib/components/Banner.svelte';
import { banner, history } from '../src/lib/stores.js';
import Downloads from '../src/pages/Downloads.svelte';

describe('Downloads receipts', () => {
  beforeEach(() => {
    banner.set(null);
    history.set([{
      id: 'gone',
      title: 'Finished video',
      channel: 'Creator',
      quality: '1080p',
      container: 'mp4',
      filename: 'gone.mp4',
      absolutePath: '/tmp/gone.mp4',
      completedAt: new Date().toISOString(),
    }]);
    (window as any).go = {
      main: {
        App: {
          OpenFile: vi.fn(async () => {
            throw new Error('That file is no longer at this path');
          }),
          RevealInFinder: vi.fn(async () => {
            throw new Error('That file is no longer at this path');
          }),
        },
      },
    };
  });

  afterEach(() => {
    banner.set(null);
    history.set([]);
  });

  test('keeps a receipt without a File missing chip and says the path is gone on Open', async () => {
    const user = userEvent.setup();
    render(Downloads);
    render(Banner);

    expect(screen.getByText('Finished video')).toBeInTheDocument();
    expect(screen.queryByText(/File missing/i)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Open downloaded file' }));
    expect(await screen.findByText('That file is no longer at this path')).toBeInTheDocument();
  });
});
