import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    blockedServicesSchedule: vi.fn(),
    blockedServicesAll: vi.fn(),
    blockedServicesScheduleUpdate: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    addWarningToast: vi.fn(),
    createUndoToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    blockedServicesSchedule: mocks.blockedServicesSchedule,
    blockedServicesAll: mocks.blockedServicesAll,
    blockedServicesScheduleUpdate: mocks.blockedServicesScheduleUpdate,
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
    addWarningToast: mocks.addWarningToast,
    createUndoToast: mocks.createUndoToast,
}));
vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import {
    allowBlockedService,
    getBlockedServices,
    getAllBlockedServices,
    servicesState,
    updateBlockedServices,
} from 'panel/stores/services';
import { resetStore } from 'panel/__tests__/helpers/resetStore';

type Deferred<T> = {
    promise: Promise<T>;
    resolve: (value: T) => void;
    reject: (reason?: unknown) => void;
};

const deferred = <T,>(): Deferred<T> => {
    let resolve: (value: T) => void = () => {};
    let reject: (reason?: unknown) => void = () => {};
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });

    return { promise, resolve, reject };
};

describe('getBlockedServices — initialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        resetStore(servicesState, { initialized: false });
    });

    it('marks initialized after a successful request', async () => {
        mocks.blockedServicesSchedule.mockResolvedValue({});

        await getBlockedServices();

        expect(servicesState.initialized).toBe(true);
    });

    it('marks initialized after a failed request so the loader cannot get stuck', async () => {
        mocks.blockedServicesSchedule.mockRejectedValue(new Error('network'));

        await getBlockedServices();

        expect(servicesState.initialized).toBe(true);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });
});

describe('getAllBlockedServices — allInitialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        resetStore(servicesState, { allInitialized: false });
    });

    it('marks allInitialized after a successful request', async () => {
        mocks.blockedServicesAll.mockResolvedValue({});

        await getAllBlockedServices();

        expect(servicesState.allInitialized).toBe(true);
    });

    it('marks allInitialized after a failed request so the loader cannot get stuck', async () => {
        mocks.blockedServicesAll.mockRejectedValue(new Error('network'));

        await getAllBlockedServices();

        expect(servicesState.allInitialized).toBe(true);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });
});

// Reset through the store itself: `resetStore` writes the raw object, which
// leaves already-created Solid nodes holding the previous value.
const seedList = async (ids: string[]) => {
    mocks.blockedServicesSchedule.mockResolvedValueOnce({ ids });
    await getBlockedServices();
    mocks.blockedServicesSchedule.mockClear();
    mocks.blockedServicesScheduleUpdate.mockClear();
};

const resetApiMocks = () => {
    vi.clearAllMocks();
    mocks.blockedServicesSchedule.mockReset();
    mocks.blockedServicesScheduleUpdate.mockReset();
    mocks.blockedServicesAll.mockReset();
};

describe('updateBlockedServices — whole-list serialization', () => {
    beforeEach(async () => {
        resetApiMocks();
        await seedList(['steam']);
    });

    it('holds processingSet across the follow-up read and applies the accepted list', async () => {
        const put = deferred<void>();
        const followUp = deferred<{ ids: string[] }>();
        mocks.blockedServicesScheduleUpdate.mockReturnValueOnce(put.promise);
        mocks.blockedServicesSchedule.mockReturnValueOnce(followUp.promise);

        const update = updateBlockedServices({ ids: ['telegram'] });

        expect(servicesState.processingSet).toBe(true);
        put.resolve();
        await vi.waitFor(() => expect(mocks.blockedServicesSchedule).toHaveBeenCalledTimes(1));
        expect(servicesState.processingSet).toBe(true);
        expect(servicesState.list.ids).toEqual(['telegram']);

        followUp.resolve({ ids: ['telegram'] });
        await update;

        expect(servicesState.processingSet).toBe(false);
        expect(servicesState.processing).toBe(false);
        expect(servicesState.list.ids).toEqual(['telegram']);
        expect(mocks.blockedServicesScheduleUpdate).toHaveBeenCalledTimes(1);
    });

    it('keeps the accepted list when the follow-up read fails', async () => {
        mocks.blockedServicesScheduleUpdate.mockResolvedValue(undefined);
        mocks.blockedServicesSchedule.mockRejectedValueOnce(new Error('get failed'));

        const ok = await updateBlockedServices({ ids: ['telegram'] });

        expect(ok).toBe(true);
        expect(servicesState.list.ids).toEqual(['telegram']);
        expect(servicesState.processingSet).toBe(false);
        expect(servicesState.processing).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalled();
    });

    it('clears both flags when the update itself fails', async () => {
        mocks.blockedServicesScheduleUpdate.mockRejectedValueOnce(new Error('put failed'));

        const ok = await updateBlockedServices({ ids: ['telegram'] });

        expect(ok).toBe(false);
        expect(servicesState.processingSet).toBe(false);
        expect(servicesState.processing).toBe(false);
        expect(mocks.addErrorToast).toHaveBeenCalled();
        expect(mocks.blockedServicesSchedule).not.toHaveBeenCalled();
    });

    it('refuses a concurrent update, warns, and sends one request', async () => {
        const put = deferred<void>();
        const followUp = deferred<{ ids: string[] }>();
        mocks.blockedServicesScheduleUpdate.mockReturnValueOnce(put.promise);
        mocks.blockedServicesSchedule.mockReturnValueOnce(followUp.promise);

        const first = updateBlockedServices({ ids: ['telegram'] });
        const second = await updateBlockedServices({ ids: ['whatsapp'] });

        expect(second).toBe(false);
        expect(mocks.addWarningToast).toHaveBeenCalledWith({
            error: expect.any(String),
        });
        expect(mocks.blockedServicesScheduleUpdate).toHaveBeenCalledTimes(1);

        put.resolve();
        await vi.waitFor(() => expect(mocks.blockedServicesSchedule).toHaveBeenCalledTimes(1));
        followUp.resolve({ ids: ['telegram'] });
        await first;
    });

    it('discards a read that started before the update', async () => {
        const staleRead = deferred<{ ids: string[] }>();
        mocks.blockedServicesSchedule.mockReturnValueOnce(staleRead.promise);
        const read = getBlockedServices();

        const put = deferred<void>();
        const followUp = deferred<{ ids: string[] }>();
        mocks.blockedServicesScheduleUpdate.mockReturnValueOnce(put.promise);
        mocks.blockedServicesSchedule.mockReturnValueOnce(followUp.promise);
        const update = updateBlockedServices({ ids: ['telegram'] });

        put.resolve();
        await vi.waitFor(() => expect(mocks.blockedServicesSchedule).toHaveBeenCalledTimes(2));
        followUp.resolve({ ids: ['telegram'] });
        await update;

        staleRead.resolve({ ids: ['steam'] });
        await read;

        expect(servicesState.list.ids).toEqual(['telegram']);
        expect(servicesState.processing).toBe(false);
    });
});

describe('allowBlockedService — shared whole-list update', () => {
    beforeEach(async () => {
        resetApiMocks();
        await seedList(['telegram']);
    });

    it('refuses to update while another whole-list update is pending', async () => {
        const put = deferred<void>();
        const followUp = deferred<{ ids: string[] }>();
        mocks.blockedServicesScheduleUpdate.mockReturnValueOnce(put.promise);
        mocks.blockedServicesSchedule.mockReturnValueOnce(followUp.promise);

        const first = updateBlockedServices({ ids: ['telegram'] });
        const ok = await allowBlockedService('telegram');

        expect(ok).toBe(false);
        expect(mocks.addWarningToast).toHaveBeenCalledWith({
            error: expect.any(String),
        });
        expect(mocks.blockedServicesScheduleUpdate).toHaveBeenCalledTimes(1);

        put.resolve();
        await vi.waitFor(() => expect(mocks.blockedServicesSchedule).toHaveBeenCalledTimes(1));
        followUp.resolve({ ids: ['telegram'] });
        await first;
    });
});
