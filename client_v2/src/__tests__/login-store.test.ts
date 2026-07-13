import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock the API client before importing the store.
const mockLogin = vi.fn().mockResolvedValue(undefined);
vi.mock('panel/api/Api', () => ({
    apiClient: {
        login: (...args: unknown[]) => mockLogin(...args),
    },
}));

// Mock addErrorToast so it does not depend on other stores.
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
}));

// Mock HTML_PAGES constant.
vi.mock('panel/helpers/constants', () => ({
    HTML_PAGES: { LOGIN: '/login.html', MAIN: '/dashboard.html' },
}));

import { processLogin } from '../stores/login';

describe('processLogin', () => {
    beforeEach(() => {
        mockLogin.mockClear();
        // Prevent window.location.replace from throwing in tests.
        Object.defineProperty(window, 'location', {
            value: { replace: vi.fn(), origin: 'http://localhost', pathname: '/login.html' },
            writable: true,
        });
    });

    it('sends { name, password } to the API (not username)', async () => {
        await processLogin({ name: 'admin', password: 'supersecret' });

        expect(mockLogin).toHaveBeenCalledTimes(1);
        expect(mockLogin).toHaveBeenCalledWith({ name: 'admin', password: 'supersecret' });
    });

    it('does not send username field', async () => {
        await processLogin({ name: 'admin', password: 'supersecret' });

        const sentArg = mockLogin.mock.calls[0][0];
        expect(sentArg).not.toHaveProperty('username');
    });
});
