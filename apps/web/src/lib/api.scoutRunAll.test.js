import { describe, it, expect, vi, afterEach } from 'vitest';
import { scout } from './api.js';

afterEach(() => { vi.restoreAllMocks(); });

describe('scout.runAll', () => {
	it('POSTs platform=all with the generate_report flag', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true, status: 201, json: async () => ({ runIds: [1, 2] }),
		});
		vi.stubGlobal('fetch', fetchMock);

		const res = await scout.runAll('p1', { generateReport: false });

		expect(res).toEqual({ runIds: [1, 2] });
		const [url, opts] = fetchMock.mock.calls[0];
		expect(url).toContain('/api/projects/p1/scout');
		expect(opts.method).toBe('POST');
		expect(JSON.parse(opts.body)).toEqual({ platform: 'all', generate_report: false });
	});

	it('defaults generate_report to true', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true, status: 201, json: async () => ({ runIds: [1] }),
		});
		vi.stubGlobal('fetch', fetchMock);

		await scout.runAll('p1');

		const [, opts] = fetchMock.mock.calls[0];
		expect(JSON.parse(opts.body).generate_report).toBe(true);
	});
});
