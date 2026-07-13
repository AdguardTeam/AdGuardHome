import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock the API client.
const mockCheckConfig = vi.fn();
vi.mock('panel/api/Api', () => ({
    apiClient: {
        checkConfig: (...args: unknown[]) => mockCheckConfig(...args),
        getDefaultAddresses: vi.fn(),
        setAllSettings: vi.fn(),
    },
}));

vi.mock('panel/stores/toasts', () => ({ addErrorToast: vi.fn() }));
vi.mock('panel/helpers/constants', () => ({
    ALL_INTERFACES_IP: '0.0.0.0',
    INSTALL_FIRST_STEP: 1,
    STANDARD_DNS_PORT: 53,
    STANDARD_WEB_PORT: 80,
}));

import { checkConfig, installState } from '../stores/install';

describe('checkConfig', () => {
    beforeEach(() => {
        mockCheckConfig.mockReset();
    });

    it('persists the request web/dns ip and port to the store', async () => {
        // Simulate the API response — only status and can_autofix, no ip/port.
        mockCheckConfig.mockResolvedValue({
            web: { status: '', can_autofix: false },
            dns: { status: '', can_autofix: false },
            static_ip: { static: 'no', ip: '', error: '' },
        });

        // User changed web port to 8080 and dns port to 5353.
        await checkConfig({
            web: { ip: '192.168.1.1', port: 8080 },
            dns: { ip: '192.168.1.1', port: 5353 },
            set_static_ip: false,
        });

        expect(installState.web.ip).toBe('192.168.1.1');
        expect(installState.web.port).toBe(8080);
        expect(installState.dns.ip).toBe('192.168.1.1');
        expect(installState.dns.port).toBe(5353);
    });

    it('does not lose ip/port when the API returns only status fields', async () => {
        mockCheckConfig.mockResolvedValue({
            web: { status: 'address already in use', can_autofix: true },
            dns: { status: '', can_autofix: false },
            static_ip: { static: 'no', ip: '', error: '' },
        });

        await checkConfig({
            web: { ip: '0.0.0.0', port: 3000 },
            dns: { ip: '0.0.0.0', port: 53 },
            set_static_ip: false,
        });

        // Port must still be the user's value, not the old store default.
        expect(installState.web.port).toBe(3000);
        expect(installState.web.status).toBe('address already in use');
        expect(installState.web.can_autofix).toBe(true);
    });
});
