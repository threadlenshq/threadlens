import { describe, it, expect, vi, afterEach } from 'vitest';
import { generalQueries } from './api.js';

afterEach(() => { vi.restoreAllMocks(); });

describe('generalQueries api', () => {
  it('POSTs to the general-queries endpoint with a JSON body', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true, status: 201, json: async () => ({ id: 1 }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await generalQueries.create('p1', { query_text: 'x', angle: 'y', platforms: ['reddit'] });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, opts] = fetchMock.mock.calls[0];
    expect(url).toContain('/api/projects/p1/general-queries');
    expect(opts.method).toBe('POST');
    expect(JSON.parse(opts.body)).toEqual({ query_text: 'x', angle: 'y', platforms: ['reddit'] });
  });
});
