import { render, waitFor } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, it, vi } from 'vitest';

import { ConfigureBlocklistModal } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal';
import { MODAL_TYPE } from 'panel/helpers/constants';

const mocks = vi.hoisted(() => ({
    closeModal: vi.fn(),
    filteringAddURL: vi.fn(),
    filteringStatus: vi.fn(() => Promise.resolve({ filters: [], whitelist_filters: [] })),
}));

vi.mock('panel/api/generated', () => ({
    filteringAddURL: mocks.filteringAddURL,
    filteringCheckHost: vi.fn(),
    filteringConfig: vi.fn(),
    filteringRefresh: vi.fn(),
    filteringRemoveURL: vi.fn(),
    filteringSetRules: vi.fn(),
    filteringSetURL: vi.fn(),
    filteringStatus: mocks.filteringStatus,
}));

vi.mock('panel/common/ui/ModalWrapper', () => ({
    ModalWrapper: (props: { children: unknown }) => <>{props.children}</>,
}));

vi.mock('panel/common/ui/Dialog/Dialog', () => ({
    Dialog: (props: { children: unknown }) => <>{props.children}</>,
}));

vi.mock('panel/common/ui/Tabs', () => ({
    Tabs: (props: { tabs: Array<{ content: unknown }> }) => <>{props.tabs[1].content}</>,
}));

vi.mock('panel/stores/modals', () => ({
    closeModal: mocks.closeModal,
    modalsState: { modalId: null },
    openModal: vi.fn(),
}));

const createDeferred = <T,>() => {
    let resolve = (_value: T) => {};
    const promise = new Promise<T>((promiseResolve) => {
        resolve = promiseResolve;
    });

    return { promise, resolve };
};

beforeEach(() => {
    vi.clearAllMocks();
});

it('keeps the manual blocklist modal open while the add request is pending', async () => {
    const addRequest = createDeferred<void>();
    const statusRequest = createDeferred<{ filters: []; whitelist_filters: [] }>();
    mocks.filteringAddURL.mockReturnValueOnce(addRequest.promise);
    mocks.filteringStatus.mockReturnValueOnce(statusRequest.promise);

    const user = userEvent.setup();
    const { container } = render(() => (
        <ConfigureBlocklistModal modalId={MODAL_TYPE.ADD_BLOCKLIST} />
    ));

    const nameInput = container.querySelector<HTMLInputElement>('input[name="name"]');
    const urlInput = container.querySelector<HTMLInputElement>('input[name="url"]');
    const submit = container.querySelector<HTMLButtonElement>('#filters_save');
    if (nameInput === null || urlInput === null || submit === null) {
        throw new Error('expected the manual blocklist form');
    }

    await user.type(nameInput, 'Slow list');
    await user.type(urlInput, 'https://example.org/slow.txt');
    await user.click(submit);

    expect(mocks.filteringAddURL).toHaveBeenCalledTimes(1);
    expect(mocks.closeModal).not.toHaveBeenCalled();
    expect(submit).toBeDisabled();

    addRequest.resolve();
    await waitFor(() => expect(mocks.filteringStatus).toHaveBeenCalledTimes(1));
    expect(mocks.closeModal).not.toHaveBeenCalled();
    expect(submit).toBeDisabled();

    statusRequest.resolve({ filters: [], whitelist_filters: [] });
    await waitFor(() => expect(mocks.closeModal).toHaveBeenCalledTimes(1));
});
