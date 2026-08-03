import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
    cancelQueryLogRequests,
    getAdditionalLogs,
    setFilteredLogs,
    queryLogsState,
} from 'panel/stores/queryLogs';
import { queryLog } from 'panel/api/generated';
import { addErrorToast } from 'panel/stores/toasts';

vi.mock('panel/api/generated', () => ({
    queryLog: vi.fn(),
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

const createDeferred = <T>() => {
    let resolve: (value: T) => void = () => {};
    let reject: (reason?: unknown) => void = () => {};
    const promise = new Promise<T>((promiseResolve, promiseReject) => {
        resolve = promiseResolve;
        reject = promiseReject;
    });

    return { promise, reject, resolve };
};

const filter = (search: string) => ({ search, status: 'rewritten', reason: 'all' });

const response = (domain: string, oldest = '') => ({
    data: [{ reason: 'Rewrite' as const, question: { name: domain } }],
    oldest,
});

describe('queryLogs store', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('getAdditionalLogs appends filtered rows using the oldest cursor and filter', async () => {
        (queryLog as any).mockReset();

        // Seed state: a full first page (20 rewritten) with more behind it.
        const fullPage = Array.from({ length: 20 }, () => ({
            reason: 'Rewrite',
            question: {},
        }));
        (queryLog as any).mockResolvedValueOnce({
            data: fullPage,
            oldest: 'cur',
        });
        await setFilteredLogs({ search: '', status: 'rewritten', reason: 'all' });
        expect(queryLogsState.isEntireLog).toBe(false);
        expect(queryLogsState.logs).toHaveLength(20);

        // Load more: one extra rewritten entry, then end of log.
        (queryLog as any).mockResolvedValueOnce({
            data: [{ reason: 'Rewrite', question: {} }],
            oldest: '',
        });

        await getAdditionalLogs();

        // The load-more request must carry the cursor and the reason filter.
        expect(queryLog).toHaveBeenLastCalledWith(
            expect.objectContaining({
                older_than: 'cur',
                reason: expect.arrayContaining([
                    'Rewrite',
                    'RewriteEtcHosts',
                    'RewriteRule',
                    'FilteredSafeSearch',
                ]),
            }),
            { signal: expect.any(AbortSignal) },
        );
        // Must NOT send the deprecated response_status
        const lastCall = (queryLog as any).mock.calls.at(-1)[0];
        expect(lastCall).not.toHaveProperty('response_status');

        expect(queryLogsState.logs).toHaveLength(21);
        expect(queryLogsState.isEntireLog).toBe(true);
        expect(queryLogsState.processingAdditionalLogs).toBe(false);
    });

    it('setFilteredLogs sends reason strings for blocked status', async () => {
        (queryLog as any).mockReset();
        (queryLog as any).mockResolvedValue({
            data: [{ reason: 'FilteredBlackList', question: {} }],
            oldest: '',
        });

        await setFilteredLogs({ search: '', status: 'blocked', reason: 'all' });

        const lastCall = (queryLog as any).mock.calls.at(-1)[0];
        expect(queryLog).toHaveBeenLastCalledWith(
            expect.objectContaining({
                reason: expect.arrayContaining([
                    'FilteredBlackList',
                    'FilteredSafeBrowsing',
                    'FilteredParental',
                    'FilteredBlockedService',
                ]),
            }),
            { signal: expect.any(AbortSignal) },
        );
        expect(lastCall).not.toHaveProperty('response_status');
    });

    it('does not mark the log as complete when additional loading stops', async () => {
        (queryLog as any).mockResolvedValue({
            data: [],
            oldest: 'next-cursor',
        });

        await getAdditionalLogs();

        expect(queryLogsState.processingAdditionalLogs).toBe(false);
        expect(queryLogsState.isEntireLog).toBe(false);
    });

    it('accumulates pages until oldest is empty (short-polling)', async () => {
        (queryLog as any)
            .mockResolvedValueOnce({
                data: [{ reason: 'Rewrite' }],
                oldest: 'cur',
                is_entire_log: false,
            })
            .mockResolvedValueOnce({
                data: [{ reason: 'Rewrite' }],
                oldest: '',
                is_entire_log: true,
            });

        await setFilteredLogs({ search: '', status: 'rewritten', reason: 'all' });
        expect(queryLog).toHaveBeenCalledTimes(2);
        expect(queryLogsState.processingGetLogs).toBe(false);
    });

    it('always sends limit=20 to prevent loading all records at once', async () => {
        (queryLog as any).mockReset();
        (queryLog as any)
            .mockResolvedValueOnce({
                data: Array.from({ length: 20 }, () => ({ reason: 'Rewrite', question: {} })),
                oldest: 'cursor1',
            })
            .mockResolvedValueOnce({
                data: [{ reason: 'Rewrite', question: {} }],
                oldest: '',
            });

        await setFilteredLogs({ search: '', status: 'rewritten', reason: 'all' });

        for (const call of (queryLog as any).mock.calls) {
            expect(call[0]).toHaveProperty('limit', 20);
        }
        expect(queryLog).not.toHaveBeenCalledWith(expect.not.objectContaining({ limit: 20 }));

        (queryLog as any).mockReset();
        (queryLog as any).mockResolvedValueOnce({
            data: [{ reason: 'Rewrite', question: {} }],
            oldest: '',
        });

        await getAdditionalLogs();

        expect((queryLog as any).mock.calls.at(-1)[0]).toEqual(
            expect.objectContaining({ limit: 20 }),
        );
    });

    it('keeps the latest filtered response when an older request finishes last', async () => {
        const older = createDeferred<ReturnType<typeof response>>();
        const latest = createDeferred<ReturnType<typeof response>>();
        (queryLog as any).mockImplementation((params: { search: string }) =>
            params.search === 'older' ? older.promise : latest.promise,
        );

        const olderRequest = setFilteredLogs(filter('older'));
        const olderSignal = (queryLog as any).mock.calls[0][1]?.signal as AbortSignal | undefined;
        const latestRequest = setFilteredLogs(filter('latest'));

        latest.resolve(response('latest.example'));
        await expect(latestRequest).resolves.toBe(true);
        expect(queryLogsState.logs[0]?.domain).toBe('latest.example');

        older.resolve(response('older.example'));
        await expect(olderRequest).resolves.toBe(false);

        expect(olderSignal?.aborted).toBe(true);
        expect(queryLogsState.filter.search).toBe('latest');
        expect(queryLogsState.logs[0]?.domain).toBe('latest.example');
    });

    it('does not show an error toast when a filtered request is aborted', async () => {
        const older = createDeferred<ReturnType<typeof response>>();
        const latest = createDeferred<ReturnType<typeof response>>();
        let olderSignal: AbortSignal | undefined;
        (queryLog as any).mockImplementation(
            (params: { search: string }, options?: { signal?: AbortSignal }) => {
                if (params.search !== 'older') {
                    return latest.promise;
                }

                olderSignal = options?.signal;
                olderSignal?.addEventListener('abort', () => {
                    older.reject(new DOMException('The operation was aborted.', 'AbortError'));
                });

                return older.promise;
            },
        );

        const olderRequest = setFilteredLogs(filter('older'));
        const latestRequest = setFilteredLogs(filter('latest'));
        older.reject(new DOMException('The operation was aborted.', 'AbortError'));

        await expect(olderRequest).resolves.toBe(false);
        expect(olderSignal?.aborted).toBe(true);
        expect(addErrorToast).not.toHaveBeenCalled();

        latest.resolve(response('latest.example'));
        await expect(latestRequest).resolves.toBe(true);
    });

    it('stops an aborted filtered short-poll chain before requesting another page', async () => {
        const olderSecondPage = createDeferred<ReturnType<typeof response>>();
        const olderCalls: Array<{ options?: { signal?: AbortSignal }; olderThan: string }> = [];
        (queryLog as any).mockImplementation(
            (params: { older_than: string; search: string }, options?: { signal?: AbortSignal }) => {
                if (params.search === 'latest') {
                    return Promise.resolve(response('latest.example'));
                }

                olderCalls.push({ options, olderThan: params.older_than });
                if (params.older_than === '') {
                    return Promise.resolve(response('older-first.example', 'cursor-1'));
                }
                if (params.older_than === 'cursor-1') {
                    return olderSecondPage.promise;
                }

                return Promise.resolve(response('unexpected.example'));
            },
        );

        const olderRequest = setFilteredLogs(filter('older'));
        await vi.waitFor(() => expect(olderCalls).toHaveLength(2));

        await expect(setFilteredLogs(filter('latest'))).resolves.toBe(true);
        olderSecondPage.resolve(response('older-second.example', 'cursor-2'));
        await expect(olderRequest).resolves.toBe(false);

        expect(olderCalls).toHaveLength(2);
        expect(olderCalls[0].options?.signal).toBeInstanceOf(AbortSignal);
        expect(olderCalls[1].options?.signal).toBe(olderCalls[0].options?.signal);
    });

    it('keeps the loading state until the latest filtered request finishes', async () => {
        const older = createDeferred<ReturnType<typeof response>>();
        const latest = createDeferred<ReturnType<typeof response>>();
        (queryLog as any).mockImplementation((params: { search: string }) =>
            params.search === 'older' ? older.promise : latest.promise,
        );

        const olderRequest = setFilteredLogs(filter('older'));
        const latestRequest = setFilteredLogs(filter('latest'));

        older.resolve(response('older.example'));
        await expect(olderRequest).resolves.toBe(false);
        expect(queryLogsState.processingGetLogs).toBe(true);

        latest.resolve(response('latest.example'));
        await expect(latestRequest).resolves.toBe(true);
        expect(queryLogsState.processingGetLogs).toBe(false);
    });

    it('discards a pending load-more response when a new filter starts', async () => {
        (queryLog as any).mockResolvedValueOnce({
            data: Array.from({ length: 20 }, () => response('filter-a.example').data[0]),
            oldest: 'filter-a-cursor',
        });
        await setFilteredLogs(filter('filter-a'));

        const stalePage = createDeferred<ReturnType<typeof response>>();
        (queryLog as any).mockReturnValueOnce(stalePage.promise);
        const loadMoreRequest = getAdditionalLogs();
        const loadMoreSignal = (queryLog as any).mock.calls.at(-1)[1]?.signal as
            | AbortSignal
            | undefined;

        (queryLog as any).mockResolvedValueOnce(response('filter-b.example'));
        await expect(setFilteredLogs(filter('filter-b'))).resolves.toBe(true);
        expect(loadMoreSignal?.aborted).toBe(true);

        stalePage.resolve(response('stale-filter-a.example', 'stale-cursor'));
        await loadMoreRequest;

        expect(queryLogsState.filter.search).toBe('filter-b');
        expect(queryLogsState.logs).toHaveLength(1);
        expect(queryLogsState.logs[0]?.domain).toBe('filter-b.example');
        expect(queryLogsState.oldest).toBe('');
        expect(queryLogsState.processingAdditionalLogs).toBe(false);
        expect(addErrorToast).not.toHaveBeenCalled();
    });

    it('cancels pending filtered work without a toast or stale state update', async () => {
        const pending = createDeferred<ReturnType<typeof response>>();
        let signal: AbortSignal | undefined;
        (queryLog as any).mockImplementation(
            (_params: unknown, options?: { signal?: AbortSignal }) => {
                signal = options?.signal;
                signal?.addEventListener('abort', () => {
                    pending.reject(new DOMException('The operation was aborted.', 'AbortError'));
                });

                return pending.promise;
            },
        );

        const request = setFilteredLogs(filter('unmounted'));
        cancelQueryLogRequests();
        await expect(request).resolves.toBe(false);

        expect(signal?.aborted).toBe(true);
        expect(queryLogsState.processingGetLogs).toBe(false);
        expect(queryLogsState.processingAdditionalLogs).toBe(false);
        expect(addErrorToast).not.toHaveBeenCalled();
    });

    it('cancels pending load-more work without a toast or stale append', async () => {
        (queryLog as any).mockResolvedValueOnce(response('kept.example'));
        await setFilteredLogs(filter('kept'));

        const pending = createDeferred<ReturnType<typeof response>>();
        let signal: AbortSignal | undefined;
        (queryLog as any).mockImplementation(
            (_params: unknown, options?: { signal?: AbortSignal }) => {
                signal = options?.signal;
                signal?.addEventListener('abort', () => {
                    pending.reject(new DOMException('The operation was aborted.', 'AbortError'));
                });

                return pending.promise;
            },
        );

        const request = getAdditionalLogs();
        cancelQueryLogRequests();
        await request;

        expect(signal?.aborted).toBe(true);
        expect(queryLogsState.logs).toHaveLength(1);
        expect(queryLogsState.logs[0]?.domain).toBe('kept.example');
        expect(queryLogsState.processingAdditionalLogs).toBe(false);
        expect(addErrorToast).not.toHaveBeenCalled();
    });
});
