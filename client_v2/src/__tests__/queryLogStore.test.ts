import { describe, expect, it, vi, beforeEach } from 'vitest';
import { getAdditionalLogs, setFilteredLogs, queryLogsState } from 'panel/stores/queryLogs';
import { apiClient } from 'panel/api/Api';

vi.mock('panel/api/Api', () => ({
    apiClient: {
        getQueryLog: vi.fn(),
    },
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

describe('queryLogs store', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('getAdditionalLogs appends filtered rows using the oldest cursor and filter', async () => {
        (apiClient.getQueryLog as any).mockReset();

        // Seed state: a full first page (20 rewritten) with more behind it.
        const fullPage = Array.from({ length: 20 }, () => ({
            reason: 'Rewrite',
            question: {},
        }));
        (apiClient.getQueryLog as any).mockResolvedValueOnce({
            data: fullPage,
            oldest: 'cur',
        });
        await setFilteredLogs({ search: '', status: 'rewritten', reason: 'all' });
        expect(queryLogsState.isEntireLog).toBe(false);
        expect(queryLogsState.logs).toHaveLength(20);

        // Load more: one extra rewritten entry, then end of log.
        (apiClient.getQueryLog as any).mockResolvedValueOnce({
            data: [{ reason: 'Rewrite', question: {} }],
            oldest: '',
        });

        await getAdditionalLogs();

        // The load-more request must carry the cursor and the reason filter.
        expect(apiClient.getQueryLog).toHaveBeenLastCalledWith(
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
        const lastCall = (apiClient.getQueryLog as any).mock.calls.at(-1)[0];
        expect(lastCall).not.toHaveProperty('response_status');

        expect(queryLogsState.logs).toHaveLength(21);
        expect(queryLogsState.isEntireLog).toBe(true);
        expect(queryLogsState.processingAdditionalLogs).toBe(false);
    });

    it('setFilteredLogs sends reason strings for blocked status', async () => {
        (apiClient.getQueryLog as any).mockReset();
        (apiClient.getQueryLog as any).mockResolvedValue({
            data: [{ reason: 'FilteredBlackList', question: {} }],
            oldest: '',
        });

        await setFilteredLogs({ search: '', status: 'blocked', reason: 'all' });

        const lastCall = (apiClient.getQueryLog as any).mock.calls.at(-1)[0];
        expect(apiClient.getQueryLog).toHaveBeenLastCalledWith(
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
        (apiClient.getQueryLog as any).mockResolvedValue({
            data: [],
            oldest: 'next-cursor',
        });

        await getAdditionalLogs();

        expect(queryLogsState.processingAdditionalLogs).toBe(false);
        expect(queryLogsState.isEntireLog).toBe(false);
    });

    it('accumulates pages until oldest is empty (short-polling)', async () => {
        (apiClient.getQueryLog as any)
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
        expect(apiClient.getQueryLog).toHaveBeenCalledTimes(2);
        expect(queryLogsState.processingGetLogs).toBe(false);
    });
});
