import { describe, expect, it } from 'vitest';

import { buildClientConfig } from 'panel/stores/clientForm';
import { getInitialClientFormState } from 'panel/initialState';

describe('buildClientConfig', () => {
    it('maps use_global_blocked_services from form.use_global_blocked_services', () => {
        const form = {
            ...getInitialClientFormState(),
            use_global_settings: true,
            use_global_blocked_services: false,
        };
        const config = buildClientConfig(form);
        expect(config.use_global_blocked_services).toBe(false);
        expect(config.use_global_settings).toBe(true);
    });

    it('maps use_global_blocked_services=true correctly', () => {
        const form = {
            ...getInitialClientFormState(),
            use_global_settings: false,
            use_global_blocked_services: true,
        };
        const config = buildClientConfig(form);
        expect(config.use_global_blocked_services).toBe(true);
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
});
