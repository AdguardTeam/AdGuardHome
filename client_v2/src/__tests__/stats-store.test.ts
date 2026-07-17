import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    stats: vi.fn(),
    clientsSearch: vi.fn(),
    addErrorToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
        stats: mocks.stats,
        clientsSearch: mocks.clientsSearch,
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
}));

import { getStats, statsState } from 'panel/stores/stats';

describe('getStats', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.clientsSearch.mockResolvedValue([]);
    });

    it('enriches top clients and stores normalizedTopClients (FR-001/019)', async () => {
        mocks.stats.mockResolvedValue({
            top_clients: [{ '1.2.3.4': 5 }],
            avg_processing_time: 0.012,
            top_blocked_domains: [],
            top_queried_domains: [],
            top_upstreams_avg_time: [],
            top_upstreams_responses: [],
        });

        await getStats();

        // searchClients was called with discovered client ids
        expect(mocks.clientsSearch).toHaveBeenCalledWith({
            clients: [{ id: '1.2.3.4' }],
        });
        // normalizedTopClients is populated (configured bucket carries info name)
        expect(Object.keys(statsState.normalizedTopClients.auto)).toContain('1.2.3.4');
    });

    it('converts avg_processing_time to milliseconds, falsy passthrough (FR-002)', async () => {
        mocks.stats.mockResolvedValue({ avg_processing_time: 0.012 });
        await getStats();
        expect(statsState.avgProcessingTime).toBe(12);

        mocks.stats.mockResolvedValue({ avg_processing_time: 0 });
        await getStats();
        expect(statsState.avgProcessingTime).toBe(0); // not NaN
    });
});
