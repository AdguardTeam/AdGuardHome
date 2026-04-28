import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import apiClient from '../api/Api';
import { HTML_PAGES } from '../helpers/constants';
import { fetchRequest } from '../api/fetch';

describe('fetchRequest', () => {
    beforeEach(() => {
        vi.stubGlobal('fetch', vi.fn());
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
    });

    test('serializes JSON request bodies and parses JSON responses', async () => {
        const fetchMock = vi.mocked(fetch);
        fetchMock.mockResolvedValue(
            new Response(JSON.stringify({ running: true }), {
                status: 200,
                headers: { 'Content-Type': 'application/json' },
            }),
        );

        const response = await fetchRequest('/control/status', 'POST', {
            data: { enabled: true },
        });

        expect(fetchMock).toHaveBeenCalledTimes(1);
        const [, init] = fetchMock.mock.calls[0];

        expect(init?.method).toBe('POST');
        expect(init?.body).toBe(JSON.stringify({ enabled: true }));
        expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json');
        expect(response).toStrictEqual({
            data: { running: true },
            status: 200,
        });
    });

    test('attaches status and response data to HTTP failures', async () => {
        const fetchMock = vi.mocked(fetch);
        fetchMock.mockResolvedValue(new Response('forbidden', { status: 403 }));

        await expect(fetchRequest('/control/profile', 'GET')).rejects.toMatchObject({
            response: {
                data: 'forbidden',
                status: 403,
            },
        });
    });
});

describe('apiClient.makeRequest', () => {
    beforeEach(() => {
        vi.stubGlobal('fetch', vi.fn());
        vi.spyOn(console, 'error').mockImplementation(() => undefined);
        window.history.replaceState({}, '', '/#dashboard');
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        vi.restoreAllMocks();
    });

    test('redirects to the login page on 403 outside auth pages', async () => {
        const fetchMock = vi.mocked(fetch);

        fetchMock.mockResolvedValue(new Response('forbidden', { status: 403 }));

        const result = await apiClient.makeRequest('profile', 'GET');

        expect(result).toBe(false);
        expect(window.location.pathname).not.toBe(HTML_PAGES.LOGIN);
    });
});
