import { describe, expect, it } from 'vitest';

import { buildClientConfig } from 'panel/stores/clientForm';
import { getInitialClientFormState } from 'panel/initialState';

describe('buildClientConfig', () => {
    it('syncs use_global_blocked_services from use_global_settings=true', () => {
        const form = {
            ...getInitialClientFormState(),
            use_global_settings: true,
            use_global_blocked_services: false,
        };
        const config = buildClientConfig(form);
        expect(config.use_global_blocked_services).toBe(true);
        expect(config.use_global_settings).toBe(true);
    });

    it('syncs use_global_blocked_services from use_global_settings=false', () => {
        const form = {
            ...getInitialClientFormState(),
            use_global_settings: false,
            use_global_blocked_services: true,
        };
        const config = buildClientConfig(form);
        expect(config.use_global_blocked_services).toBe(false);
        expect(config.use_global_settings).toBe(false);
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
