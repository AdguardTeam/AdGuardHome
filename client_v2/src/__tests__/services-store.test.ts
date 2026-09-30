import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    blockedServicesSchedule: vi.fn(),
    blockedServicesAll: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    createUndoToast: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    blockedServicesSchedule: mocks.blockedServicesSchedule,
    blockedServicesAll: mocks.blockedServicesAll,
    blockedServicesScheduleUpdate: vi.fn(),
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
    createUndoToast: mocks.createUndoToast,
}));
vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import { getBlockedServices, getAllBlockedServices, servicesState } from 'panel/stores/services';

describe('getBlockedServices — initialized flag', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        servicesState.initialized = false;
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
        servicesState.allInitialized = false;
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
