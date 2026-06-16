import { describe, expect, it, vi, beforeEach } from 'vitest';
import { getAdditionalLogs, queryLogsState } from 'panel/stores/queryLogs';
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

    it('does not mark the log as complete when additional loading stops', async () => {
        // Mock the API to return a non-complete log
        (apiClient.getQueryLog as any).mockResolvedValue({
            data: [],
            oldest: '',
            is_entire_log: false,
        });

        // Simulate the state before additional loading
        // The store starts with processingAdditionalLogs: false
        await getAdditionalLogs();

        expect(queryLogsState.processingAdditionalLogs).toBe(false);
        expect(queryLogsState.isEntireLog).toBe(false);
    });
});
