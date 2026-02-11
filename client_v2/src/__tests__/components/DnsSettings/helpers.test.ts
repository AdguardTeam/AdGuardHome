import { describe, expect, it, vi } from 'vitest';

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: vi.fn((key: string) => key),
    },
}));

import {
    getBlockingModeOptions,
    getBlockingModeSummary,
} from 'panel/components/DnsSettings/helpers';
import { BLOCKING_MODES } from 'panel/helpers/constants';

describe('DNS settings helpers', () => {
    it('exposes the NOERROR blocking mode', () => {
        expect(getBlockingModeSummary(BLOCKING_MODES.noerror)).toBe('NOERROR');
        expect(
            getBlockingModeOptions().find(({ value }) => value === BLOCKING_MODES.noerror),
        ).toEqual({
            text: 'dns_blocking_mode_noerror',
            value: BLOCKING_MODES.noerror,
            description: 'dns_blocking_mode_noerror_desc',
        });
    });
});
