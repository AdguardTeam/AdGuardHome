import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { CHECK_TIMEOUT } from '../helpers/constants';
import { checkRedirect } from '../helpers/helpers';

describe('checkRedirect', () => {
    beforeEach(() => {
        vi.stubGlobal('fetch', vi.fn());
        vi.spyOn(console, 'error').mockImplementation(() => undefined);
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    test('redirects after any HTTP response, including non-2xx', async () => {
        const fetchMock = vi.mocked(fetch);
        fetchMock.mockResolvedValue(new Response('', { status: 503 }));

        checkRedirect('https://example.org/login');

        await Promise.resolve();

        expect(fetchMock).toHaveBeenCalledWith('https://example.org/login');
        expect(console.error).toHaveBeenCalled();
    });

    test('retries after a network failure and redirects on the next response', async () => {
        vi.useFakeTimers();
        const fetchMock = vi.mocked(fetch);

        fetchMock.mockRejectedValueOnce(new TypeError('network error'));
        fetchMock.mockResolvedValueOnce(new Response('', { status: 200 }));

        checkRedirect('https://example.org/login');

        await Promise.resolve();
        expect(console.error).not.toHaveBeenCalled();

        await vi.advanceTimersByTimeAsync(CHECK_TIMEOUT);
        await Promise.resolve();

        expect(fetchMock).toHaveBeenCalledTimes(2);
        expect(console.error).toHaveBeenCalled();
    });
});
