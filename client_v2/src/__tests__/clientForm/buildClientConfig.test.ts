import { describe, expect, it } from 'vitest';

import { buildClientConfig, buildFormPayload } from 'panel/stores/clientForm';
import { getInitialClientFormState } from 'panel/initialState';

describe('buildClientConfig', () => {
    it('keeps use_global_blocked_services independent of use_global_settings', () => {
        const form = {
            ...getInitialClientFormState(),
            use_global_settings: true,
            use_global_blocked_services: false,
        };
        const config = buildClientConfig(form);
        expect(config.use_global_blocked_services).toBe(false);
        expect(config.use_global_settings).toBe(true);
    });

    it('keeps use_global_blocked_services when use_global_settings is off', () => {
        const form = {
            ...getInitialClientFormState(),
            use_global_settings: false,
            use_global_blocked_services: true,
        };
        const config = buildClientConfig(form);
        expect(config.use_global_blocked_services).toBe(true);
        expect(config.use_global_settings).toBe(false);
    });

    it('preserves client blocked services and schedule when global blocked services are enabled', () => {
        const form = {
            ...getInitialClientFormState(),
            use_global_blocked_services: true,
            blocked_services: ['youtube', 'steam'],
            blocked_services_schedule: {
                time_zone: 'UTC',
                mon: { start: 0, end: 60 },
            },
        };
        const config = buildClientConfig(form);
        expect(config.blocked_services).toEqual(['youtube', 'steam']);
        expect(config.blocked_services_schedule).toEqual({
            time_zone: 'UTC',
            mon: { start: 0, end: 60 },
        });
    });

    it('filters empty IDs', () => {
        const form = {
            ...getInitialClientFormState(),
            name: 'Test',
            ids: ['192.168.1.1', '', '  '],
        };
        const config = buildClientConfig(form);
        expect(config.ids).toEqual(['192.168.1.1']);
    });

    it('splits upstreams by newline and filters blank lines', () => {
        const form = {
            ...getInitialClientFormState(),
            upstreams: '1.1.1.1\n\n8.8.8.8\n  \n',
        };
        const config = buildClientConfig(form);
        expect(config.upstreams).toEqual(['1.1.1.1', '8.8.8.8']);
    });

    it('returns empty array when upstreams is empty string', () => {
        const form = {
            ...getInitialClientFormState(),
            upstreams: '',
        };
        const config = buildClientConfig(form);
        expect(config.upstreams).toEqual([]);
    });

    it('converts empty cache size string to 0', () => {
        const form = {
            ...getInitialClientFormState(),
            upstreams_cache_size: '',
        };
        const config = buildClientConfig(form);
        expect(config.upstreams_cache_size).toBe(0);
    });

    it('converts numeric cache size string to number', () => {
        const form = {
            ...getInitialClientFormState(),
            upstreams_cache_size: '123',
        };
        const config = buildClientConfig(form);
        expect(config.upstreams_cache_size).toBe(123);
    });

    it('returns all expected top-level keys', () => {
        const config = buildClientConfig(getInitialClientFormState());
        expect(Object.keys(config).sort()).toEqual([
            'blocked_services',
            'blocked_services_schedule',
            'filtering_enabled',
            'ids',
            'ignore_querylog',
            'ignore_statistics',
            'name',
            'parental_enabled',
            'safe_search',
            'safebrowsing_enabled',
            'tags',
            'upstreams',
            'upstreams_cache_enabled',
            'upstreams_cache_size',
            'use_global_blocked_services',
            'use_global_settings',
        ]);
    });
});

describe('buildFormPayload', () => {
    it('preserves the full blocked services schedule', () => {
        const payload = buildFormPayload({
            name: 'Laptop',
            ids: ['192.168.1.1'],
            blocked_services: ['youtube'],
            use_global_blocked_services: true,
            blocked_services_schedule: {
                time_zone: 'UTC',
                mon: { start: 0, end: 60 },
                sat: { start: 120, end: 240 },
            },
        } as any);

        expect(payload.blocked_services).toEqual(['youtube']);
        expect(payload.use_global_blocked_services).toBe(true);
        expect(payload.blocked_services_schedule).toEqual({
            time_zone: 'UTC',
            mon: { start: 0, end: 60 },
            sat: { start: 120, end: 240 },
        });
    });
});
