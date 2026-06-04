import { afterEach, describe, expect, it, vi } from 'vitest';
import { posts } from './api.js';

describe('posts API', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('passes dm=true through listPage params for the DMs-only filter', async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 200,
      json: async () => ({
        items: [],
        pagination: {
          page: 1,
          limit: 20,
          total: 0,
          totalPages: 1,
          hasPreviousPage: false,
          hasNextPage: false,
        },
      }),
    }));
    vi.stubGlobal('fetch', fetchMock);

    await posts.listPage('project-1', { status: 'new', dm: 'true', page: '1', limit: '20' });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe('/api/projects/project-1/posts?status=new&dm=true&page=1&limit=20');
  });
});
