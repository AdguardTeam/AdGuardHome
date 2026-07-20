import { describe, it, expect } from 'vitest';

import {
    buildQueryLogConfig,
    buildStatsConfig,
    type QueryLogConfig,
    type StatsConfig,
} from 'panel/components/Settings/helpers';

describe('buildQueryLogConfig', () => {
    it('returns only the five config fields', () => {
        const state: QueryLogConfig & Record<string, unknown> = {
            enabled: true,
            anonymize_client_ip: false,
            interval: 86400000,
            ignored: ['example.org'],
            ignored_enabled: true,
            // runtime fields that must NOT appear in the result
            processingGetLogs: true,
            processingClear: false,
            processingGetConfig: false,
            processingSetConfig: false,
            processingAdditionalLogs: false,
            logs: [] as unknown[],
            oldest: '',
            filter: {} as Record<string, unknown>,
            isFiltered: false,
            isDetailed: true,
            isEntireLog: false,
            customInterval: null as null,
        };
        const result = buildQueryLogConfig(state);
        expect(result).toEqual({
            enabled: true,
            anonymize_client_ip: false,
            interval: 86400000,
            ignored: ['example.org'],
            ignored_enabled: true,
        });
    });

    it('applies overrides on top of the store values', () => {
        const state: QueryLogConfig = {
            enabled: true,
            anonymize_client_ip: false,
            interval: 86400000,
            ignored: [],
            ignored_enabled: false,
        };
        const result = buildQueryLogConfig(state, {
            enabled: false,
            ignored: ['a', 'b'],
        });
        expect(result).toEqual({
            enabled: false,
            anonymize_client_ip: false,
            interval: 86400000,
            ignored: ['a', 'b'],
            ignored_enabled: false,
        });
    });
});

describe('buildStatsConfig', () => {
    it('returns only the four config fields', () => {
        const state: StatsConfig & Record<string, unknown> = {
            enabled: true,
            interval: 86400000,
            ignored: [],
            ignored_enabled: false,
            processingGetConfig: false,
            processingSetConfig: false,
            processingStats: false,
            processingReset: false,
            customInterval: null as null,
            dnsQueries: [] as unknown[],
            topClients: [] as unknown[],
            avgProcessingTime: 0,
        };
        const result = buildStatsConfig(state);
        expect(result).toEqual({
            enabled: true,
            interval: 86400000,
            ignored: [],
            ignored_enabled: false,
        });
    });

    it('applies overrides', () => {
        const state = {
            enabled: true,
            interval: 86400000,
            ignored: ['x'],
            ignored_enabled: true,
        };
        const result = buildStatsConfig(state, { enabled: false });
        expect(result.enabled).toBe(false);
        expect(result.interval).toBe(86400000);
    });
});
