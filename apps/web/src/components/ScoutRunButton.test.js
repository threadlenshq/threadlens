// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import ScoutRunButton from './ScoutRunButton.svelte';
import { scout as scoutApi } from '../lib/api.js';

vi.mock('../lib/api.js', () => ({
  scout: { run: vi.fn(), runAll: vi.fn() },
}));

// jsdom lacks the Web Animations API that Svelte 5 transitions (used by Modal) rely on.
if (typeof Element !== 'undefined' && !Element.prototype.animate) {
  Element.prototype.animate = () => ({
    cancel() {}, finish() {}, play() {}, pause() {}, reverse() {},
    addEventListener() {}, removeEventListener() {},
    finished: Promise.resolve(), currentTime: 0, startTime: 0, playState: 'finished',
  });
}

afterEach(cleanup);

async function selectPlatform(label) {
  await fireEvent.click(screen.getByTitle('Select platform'));
  await fireEvent.click(screen.getByText(label));
}

describe('ScoutRunButton no-query guidance', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scoutApi.run.mockResolvedValue({ runId: 1 });
    scoutApi.runAll.mockResolvedValue({ runIds: [1] });
  });

  it('guides to Sources and does not run when the chosen platform has no queries', async () => {
    render(ScoutRunButton, {
      projectId: 'p1',
      capabilities: null,
      enabledQueryCounts: { reddit: 2, hackernews: 0 },
    });

    await selectPlatform('Hacker News');

    expect(scoutApi.run).not.toHaveBeenCalled();
    // Notice content + the Go to Sources action are shown.
    expect(await screen.findByText('No Hacker News queries yet')).toBeTruthy();
    expect(screen.getByText('Go to Sources')).toBeTruthy();
  });

  it('runs when the chosen platform has queries', async () => {
    render(ScoutRunButton, {
      projectId: 'p1',
      capabilities: null,
      enabledQueryCounts: { reddit: 2, hackernews: 1 },
    });

    await selectPlatform('Hacker News');

    expect(scoutApi.run).toHaveBeenCalledWith('p1', 'hackernews');
  });

  it('All Platforms fires a single runAll call when any platform has queries, and guides when all are empty', async () => {
    const { rerender } = render(ScoutRunButton, {
      projectId: 'p1',
      capabilities: null,
      enabledQueryCounts: { reddit: 2, bluesky: 0, google: 0, hackernews: 0 },
    });

    // reddit has queries -> All Platforms fires a single unified run call.
    await selectPlatform('All Platforms');
    expect(scoutApi.runAll).toHaveBeenCalledTimes(1);
    expect(scoutApi.runAll).toHaveBeenCalledWith('p1', { generateReport: true });
    expect(scoutApi.run).not.toHaveBeenCalled();

    vi.clearAllMocks();
    scoutApi.runAll.mockResolvedValue({ runIds: [1] });
    await rerender({
      projectId: 'p1',
      capabilities: null,
      enabledQueryCounts: { reddit: 0, bluesky: 0, google: 0, hackernews: 0 },
    });

    // Nothing has queries -> guidance, no run.
    await selectPlatform('All Platforms');
    expect(scoutApi.runAll).not.toHaveBeenCalled();
    expect(await screen.findByText('No search queries yet')).toBeTruthy();
  });

  it('does not block when per-platform counts have not loaded yet', async () => {
    render(ScoutRunButton, {
      projectId: 'p1',
      capabilities: null,
      enabledQueryCounts: null,
    });

    await selectPlatform('Hacker News');

    expect(scoutApi.run).toHaveBeenCalledWith('p1', 'hackernews');
  });
});
