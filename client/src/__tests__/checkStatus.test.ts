import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { checkStatus } from '../actions/index';
import { CHECK_TIMEOUT } from '../helpers/constants';

describe('checkStatus', () => {
    beforeEach(() => {
        vi.stubGlobal('fetch', vi.fn());
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    test('retries after a non-200 response and then reports success', async () => {
        vi.useFakeTimers();
        const fetchMock = vi.mocked(fetch);

        fetchMock.mockResolvedValueOnce(new Response('temporary error', { status: 503 }));
        fetchMock.mockResolvedValueOnce(
            new Response(JSON.stringify({ running: true }), {
                status: 200,
                headers: { 'Content-Type': 'application/json' },
            }),
        );

        const handleRequestSuccess = vi.fn();
        const handleRequestError = vi.fn();

        checkStatus(handleRequestSuccess, handleRequestError, 2);

        await Promise.resolve();
        await vi.advanceTimersByTimeAsync(CHECK_TIMEOUT);
        await Promise.resolve();

        expect(handleRequestSuccess).toHaveBeenCalledWith({
            data: { running: true },
            status: 200,
        });
        expect(handleRequestError).not.toHaveBeenCalled();
    });

    test('stops immediately when no attempts remain', async () => {
        const handleRequestSuccess = vi.fn();
        const handleRequestError = vi.fn();

        await checkStatus(handleRequestSuccess, handleRequestError, 0);

        expect(handleRequestSuccess).not.toHaveBeenCalled();
        expect(handleRequestError).toHaveBeenCalledTimes(1);
        expect(fetch).not.toHaveBeenCalled();
    });
});
