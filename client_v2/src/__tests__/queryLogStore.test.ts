import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
    clearLogs,
    getAdditionalLogs,
    setFilteredLogs,
    getLogsConfig,
    queryLogsState,
} from 'panel/stores/queryLogs';
import { resetStore } from 'panel/__tests__/helpers/resetStore';
import { queryLog, querylogClear, getQueryLogConfig } from 'panel/api/generated';
import { addErrorToast } from 'panel/stores/toasts';

vi.mock('panel/api/generated', () => ({
    queryLog: vi.fn(),
    querylogClear: vi.fn(),
    getQueryLogConfig: vi.fn(),
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

type Deferred<T> = {
    promise: Promise<T>;
    resolve: (value: T) => void;
    reject: (reason?: unknown) => void;
};

const deferred = <T,>(): Deferred<T> => {
    let resolve: (value: T) => void = () => {};
    let reject: (reason?: unknown) => void = () => {};
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });

    return { promise, resolve, reject };
};

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

        expect(queryLog).toHaveBeenCalledWith(expect.objectContaining({ limit: 20 }));
    });
});

describe('queryLogs store — stale responses', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        (queryLog as any).mockReset();
        resetStore(queryLogsState, {
            processingGetLogs: false,
            processingAdditionalLogs: false,
            processingClear: false,
            logs: [],
            oldest: '',
            isEntireLog: false,
        });
    });

    it('keeps the newer search when the older response lands last', async () => {
        const stale = deferred<{ data: unknown[]; oldest: string }>();
        const fresh = deferred<{ data: unknown[]; oldest: string }>();
        (queryLog as any).mockReturnValueOnce(stale.promise).mockReturnValueOnce(fresh.promise);

        const olderSearch = setFilteredLogs({ search: '"', status: 'all', reason: 'all' });
        const newerSearch = setFilteredLogs({
            search: '"google.com"',
            status: 'all',
            reason: 'all',
        });

        fresh.resolve({ data: [{ reason: 'NotFilteredNotFound', question: {} }], oldest: '' });
        expect(await newerSearch).toBe(true);

        stale.resolve({ data: [], oldest: '' });
        expect(await olderSearch).toBe(false);

        expect(queryLogsState.filter.search).toBe('"google.com"');
        expect(queryLogsState.logs).toHaveLength(1);
        expect(queryLogsState.processingGetLogs).toBe(false);
    });

    it('does not clear the loading flag or toast when a superseded search fails', async () => {
        const stale = deferred<{ data: unknown[]; oldest: string }>();
        const fresh = deferred<{ data: unknown[]; oldest: string }>();
        (queryLog as any).mockReturnValueOnce(stale.promise).mockReturnValueOnce(fresh.promise);

        const olderSearch = setFilteredLogs({ search: '"', status: 'all', reason: 'all' });
        const newerSearch = setFilteredLogs({
            search: '"google.com"',
            status: 'all',
            reason: 'all',
        });

        fresh.resolve({ data: [], oldest: '' });
        await newerSearch;

        stale.reject(new Error('late failure'));
        await olderSearch;

        expect(queryLogsState.processingGetLogs).toBe(false);
        expect(addErrorToast).not.toHaveBeenCalled();
    });

    it('drops a page loaded under an older filter', async () => {
        resetStore(queryLogsState, {
            processingGetLogs: false,
            logs: [],
            oldest: 'cursor',
        });
        const stalePage = deferred<{ data: unknown[]; oldest: string }>();
        (queryLog as any).mockReturnValueOnce(stalePage.promise);
        (queryLog as any).mockResolvedValueOnce({
            data: [{ reason: 'NotFilteredNotFound', question: {} }],
            oldest: '',
        });

        const more = getAdditionalLogs();
        await setFilteredLogs({ search: '"google.com"', status: 'all', reason: 'all' });

        stalePage.resolve({ data: [{ reason: 'Rewrite', question: {} }], oldest: 'next' });
        await more;

        expect(queryLogsState.logs).toHaveLength(1);
        expect(queryLogsState.logs[0]?.reason).toBe('NotFilteredNotFound');
        expect(queryLogsState.processingAdditionalLogs).toBe(false);
    });

    it('refuses a second additional load while one is pending', async () => {
        resetStore(queryLogsState, { processingGetLogs: false, oldest: 'cursor' });
        const page = deferred<{ data: unknown[]; oldest: string }>();
        (queryLog as any).mockReturnValueOnce(page.promise);

        const first = getAdditionalLogs();
        await getAdditionalLogs();

        expect(queryLog).toHaveBeenCalledTimes(1);

        page.resolve({ data: [], oldest: '' });
        await first;
    });

    it('keeps the log empty when a search settles after a successful clear', async () => {
        const stale = deferred<{ data: unknown[]; oldest: string }>();
        (queryLog as any).mockReturnValueOnce(stale.promise);
        (querylogClear as any).mockResolvedValue(undefined);

        const search = setFilteredLogs({ search: '"', status: 'all', reason: 'all' });
        resetStore(queryLogsState, { processingAdditionalLogs: true });

        await clearLogs();

        stale.resolve({ data: [{ reason: 'FilteredBlackList', question: {} }], oldest: '' });
        await search;

        expect(queryLogsState.logs).toEqual([]);
        expect(queryLogsState.processingGetLogs).toBe(false);
        expect(queryLogsState.processingAdditionalLogs).toBe(false);
    });

    it('leaves an in-flight search alive when the clear fails', async () => {
        const pending = deferred<{ data: unknown[]; oldest: string }>();
        (queryLog as any).mockReturnValueOnce(pending.promise);
        (querylogClear as any).mockRejectedValueOnce(new Error('clear failed'));

        const search = setFilteredLogs({ search: '"', status: 'all', reason: 'all' });
        await clearLogs();

        expect(queryLogsState.processingClear).toBe(false);
        expect(queryLogsState.processingGetLogs).toBe(true);
        expect(addErrorToast).toHaveBeenCalled();

        pending.resolve({ data: [], oldest: '' });
        await search;
    });
});

describe('getLogsConfig — configInitialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        resetStore(queryLogsState, { configInitialized: false });
    });

    it('marks configInitialized after a successful request', async () => {
        (getQueryLogConfig as any).mockResolvedValue({});

        await getLogsConfig();

        expect(queryLogsState.configInitialized).toBe(true);
    });

    it('marks configInitialized after a failed request so the loader cannot get stuck', async () => {
        (getQueryLogConfig as any).mockRejectedValue(new Error('network'));

        await getLogsConfig();

        expect(queryLogsState.configInitialized).toBe(true);
    });
});
