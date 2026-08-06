import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';

import { FilterUpdateModal } from 'panel/components/FilterLists/blocks/FilterUpdateModal';
import { getFilteringStatus } from 'panel/stores/filtering';
import { openModal, closeModal } from 'panel/stores/modals';
import { MODAL_TYPE } from 'panel/helpers/constants';
import intl from 'panel/common/intl';

const mocks = vi.hoisted(() => ({
    apiSetFiltersConfig: vi.fn(() => Promise.resolve(undefined)),
    apiGetFilteringStatus: vi.fn(() => Promise.resolve({})),
}));

vi.mock('panel/api/generated', () => ({
    filteringConfig: mocks.apiSetFiltersConfig,
    filteringStatus: mocks.apiGetFilteringStatus,
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    createUndoToast: vi.fn(),
}));

const checkedIntervalIds = () =>
    (screen.getAllByRole('radio') as HTMLInputElement[])
        .filter((r) => r.checked)
        .map((r) => r.id);

const statusWithInterval = (interval: number) => ({
    enabled: true,
    interval,
    filters: [] as unknown[],
    whitelist_filters: [] as unknown[],
    clients_filters: [] as unknown[],
    user_rules: [] as string[],
});

describe('FilterUpdateModal interval', () => {
    beforeEach(async () => {
        vi.clearAllMocks();
        closeModal();
        mocks.apiGetFilteringStatus.mockResolvedValue(statusWithInterval(24));
        await getFilteringStatus();
    });

    afterEach(async () => {
        await intl.changeLanguage('en');
    });

    it('shows the disabled radio label translated', async () => {
        await intl.changeLanguage('vi');
        openModal(MODAL_TYPE.FILTER_UPDATE);
        render(() => <FilterUpdateModal />);

        expect(screen.getByText('Vô hiệu')).toBeInTheDocument();
    });

    it('shows hourly radio selected when interval is 1 hour', async () => {
        mocks.apiGetFilteringStatus.mockResolvedValue(statusWithInterval(1));
        await getFilteringStatus();

        openModal(MODAL_TYPE.FILTER_UPDATE);
        render(() => <FilterUpdateModal />);

        expect(checkedIntervalIds()).toEqual(['interval-1']);
    });

    it('submits the selected hourly interval', async () => {
        openModal(MODAL_TYPE.FILTER_UPDATE);
        const { container } = render(() => <FilterUpdateModal />);

        const hourly = container.querySelector('#interval-1') as HTMLInputElement;
        await userEvent.click(hourly);
        await userEvent.click(screen.getByRole('button', { name: 'Save' }));

        expect(mocks.apiSetFiltersConfig).toHaveBeenCalledWith({
            enabled: true,
            interval: 1,
        });
    });

    it('keeps hourly selection when status resolves to 1 hour after modal opens', async () => {
        let resolveStatus: (v: object) => void;
        mocks.apiGetFilteringStatus.mockReturnValueOnce(
            new Promise((resolve) => {
                resolveStatus = resolve;
            }) as Promise<object>,
        );

        const statusPromise = getFilteringStatus();
        openModal(MODAL_TYPE.FILTER_UPDATE);
        render(() => <FilterUpdateModal />);

        resolveStatus!(statusWithInterval(1));
        await statusPromise;

        expect(checkedIntervalIds()).toEqual(['interval-1']);
    });
});
