import { render, screen, fireEvent, within, waitFor } from '@solidjs/testing-library';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { Cache } from 'panel/components/DnsSettings/Cache';
import { CacheInputDialog } from 'panel/components/DnsSettings/Cache/blocks/CacheInputDialog';
import { getCacheSizeSummary, getTtlSummary } from 'panel/components/DnsSettings/helpers';
import { UINT32_RANGE } from 'panel/helpers/constants';
import { dnsConfigState, getDnsConfig } from 'panel/stores/dnsConfig';
import { copy, copyInDom, type CopyKey } from 'panel/__tests__/helpers/copy';

type DnsInfo = {
    cache_enabled?: boolean;
    cache_size: number;
    cache_ttl_min: number;
    cache_ttl_max: number;
    cache_optimistic: boolean;
};

const mocks = vi.hoisted(() => ({
    apiDnsInfo: vi.fn(),
    apiDnsConfig: vi.fn(),
    apiCacheClear: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    dnsInfo: mocks.apiDnsInfo,
    dnsConfig: mocks.apiDnsConfig,
    cacheClear: mocks.apiCacheClear,
}));

vi.mock('panel/stores/toasts', () => ({
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

const defaultConfig: DnsInfo = {
    cache_enabled: false,
    cache_size: 0,
    cache_ttl_min: 0,
    cache_ttl_max: 0,
    cache_optimistic: false,
};

const renderCache = async (overrides: Partial<DnsInfo> = {}) => {
    mocks.apiDnsInfo.mockResolvedValue({ ...defaultConfig, ...overrides });

    await getDnsConfig();
    render(() => <Cache />);
};

const getRow = (title: CopyKey) => {
    const label = screen
        .getAllByText(copyInDom(title))
        .find((element) => !element.closest('[role="dialog"]'));

    if (!label) {
        throw new Error(`Cache row not found: ${title}`);
    }

    const row = label.closest('[role="button"]');
    if (!(row instanceof HTMLElement)) {
        throw new Error(`Cache row is not an element: ${title}`);
    }

    return row;
};

const getDialog = (title: CopyKey) => {
    const dialogTitle = `${title}_title` as CopyKey;
    const label = screen
        .getAllByText(copyInDom(dialogTitle))
        .find((element) => element.closest('[role="dialog"]'));

    if (!label) {
        throw new Error(`Cache dialog title not found: ${title}`);
    }

    const dialog = label.closest('[role="dialog"]');
    if (!(dialog instanceof HTMLElement)) {
        throw new Error(`Cache dialog is not an element: ${title}`);
    }

    return dialog;
};

const getSwitch = (id: string) => {
    const input = document.querySelector(`input#${id}`);
    if (!(input instanceof HTMLInputElement)) {
        throw new Error(`Cache switch not found: ${id}`);
    }

    return input;
};

const openDialog = async (title: CopyKey) => {
    fireEvent.click(getRow(title));

    const dialog = getDialog(title);
    const input = await within(dialog).findByRole('spinbutton');
    if (!(input instanceof HTMLInputElement)) {
        throw new Error(`Cache dialog input not found: ${title}`);
    }

    return { dialog, input };
};

const saveDialog = (dialog: HTMLElement) => {
    fireEvent.click(within(dialog).getByTestId('config-dialog-save'));
};

const waitForSave = () => waitFor(() => expect(dnsConfigState.processingSetConfig).toBe(false));

const deferred = <T,>() => {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>((res) => {
        resolve = res;
    });

    return { promise, resolve };
};

describe('Cache', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.apiDnsConfig.mockResolvedValue(undefined);
        mocks.apiCacheClear.mockResolvedValue(undefined);
    });

    describe('rendering', () => {
        it('renders the cache settings and their current values', async () => {
            await renderCache({
                cache_enabled: true,
                cache_size: 4194304,
                cache_ttl_min: 60,
                cache_ttl_max: 3600,
                cache_optimistic: true,
            });

            expect(screen.getByText(copyInDom('dns_cache_title'))).toBeDefined();
            expect(getRow('dns_cache_size')).toBeDefined();
            expect(screen.getByText('4 MB')).toBeDefined();
            expect(screen.getByText(copyInDom('dns_ttl_value', { value: 60 }))).toBeDefined();
            expect(screen.getByText(copyInDom('dns_ttl_value', { value: 3600 }))).toBeDefined();
            expect(getSwitch('cache_enabled').checked).toBe(true);
            expect(getSwitch('optimistic_caching').checked).toBe(true);
        });

        it('disables optimistic caching while the cache is disabled', async () => {
            await renderCache({ cache_enabled: false });

            expect(getSwitch('optimistic_caching').disabled).toBe(true);
        });

        it('enables optimistic caching when the cache is enabled', async () => {
            await renderCache({ cache_enabled: true });

            expect(getSwitch('optimistic_caching').disabled).toBe(false);
        });

        it('shows a zero cache size summary', async () => {
            await renderCache({ cache_size: 0 });

            expect(screen.getByText('0 bytes')).toBeDefined();
        });

        it('disables the cache size row while the cache state is unknown', async () => {
            await renderCache({ cache_enabled: undefined });

            expect(getRow('dns_cache_size')).toHaveAttribute('tabindex', '-1');
        });
    });

    describe('value rows', () => {
        it.each(['dns_cache_size', 'dns_override_min_ttl', 'dns_override_max_ttl'] as const)(
            'allows editing %s while the cache is disabled',
            async (title) => {
                await renderCache();

                const row = getRow(title);
                expect(row).toHaveAttribute('tabindex', '0');

                fireEvent.click(row);

                const dialog = getDialog(title);
                expect(await within(dialog).findByRole('spinbutton')).toBeDefined();
            },
        );

        it('prefills the dialog with the stored value', async () => {
            await renderCache({ cache_size: 1048576 });

            const { input } = await openDialog('dns_cache_size');

            await waitFor(() => expect(input.value).toBe('1048576'));
        });
    });

    describe('saving values', () => {
        it('keeps the cache switch state when saving the size while the cache is disabled', async () => {
            await renderCache({ cache_enabled: false });

            const { dialog, input } = await openDialog('dns_cache_size');

            fireEvent.change(input, { target: { value: '4194304' } });
            saveDialog(dialog);

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({
                    cache_size: 4194304,
                    cache_enabled: false,
                }),
            );
            await waitForSave();
        });

        it('keeps the cache switch state when saving the size while the cache is enabled', async () => {
            await renderCache({ cache_enabled: true, cache_size: 1024 });

            const { dialog, input } = await openDialog('dns_cache_size');

            fireEvent.change(input, { target: { value: '1048576' } });
            saveDialog(dialog);

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({
                    cache_size: 1048576,
                    cache_enabled: true,
                }),
            );
            await waitForSave();
        });

        it('omits cache_enabled when it is unknown', async () => {
            mocks.apiDnsInfo.mockResolvedValue({
                cache_enabled: undefined,
                cache_size: 0,
                cache_ttl_min: 0,
                cache_ttl_max: 0,
                cache_optimistic: false,
            });

            await getDnsConfig();
            render(() => (
                <CacheInputDialog
                    configKey="cache_size"
                    open={() => true}
                    onClose={vi.fn()}
                    processing={false}
                />
            ));

            const dialog = getDialog('dns_cache_size');
            const input = await within(dialog).findByRole('spinbutton');

            fireEvent.change(input, { target: { value: '1048576' } });
            saveDialog(dialog);

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({ cache_size: 1048576 }),
            );
            await waitForSave();
        });

        it('saves the minimum TTL override', async () => {
            await renderCache({ cache_ttl_min: 0, cache_ttl_max: 3600 });

            const { dialog, input } = await openDialog('dns_override_min_ttl');

            fireEvent.change(input, { target: { value: '60' } });
            saveDialog(dialog);

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({ cache_ttl_min: 60 }),
            );
            await waitForSave();
        });

        it('saves the maximum TTL override', async () => {
            await renderCache({ cache_ttl_min: 0, cache_ttl_max: 0 });

            const { dialog, input } = await openDialog('dns_override_max_ttl');

            fireEvent.change(input, { target: { value: '3600' } });
            saveDialog(dialog);

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({ cache_ttl_max: 3600 }),
            );
            await waitForSave();
        });
    });

    describe('validation', () => {
        it('rejects an empty cache size', async () => {
            await renderCache({ cache_size: 1024 });

            const { dialog, input } = await openDialog('dns_cache_size');

            fireEvent.change(input, { target: { value: '' } });
            saveDialog(dialog);

            expect(mocks.apiDnsConfig).not.toHaveBeenCalled();
            expect(within(dialog).getByText(copyInDom('form_error_required'))).toBeDefined();
        });

        it('rejects a zero cache size while the cache is enabled', async () => {
            await renderCache({ cache_enabled: true, cache_size: 1024 });

            const { dialog, input } = await openDialog('dns_cache_size');

            fireEvent.change(input, { target: { value: '0' } });
            saveDialog(dialog);

            expect(mocks.apiDnsConfig).not.toHaveBeenCalled();
            expect(
                within(dialog).getByText(copyInDom('cache_config_size_validation')),
            ).toBeDefined();
        });

        it('allows a zero cache size while the cache is disabled', async () => {
            await renderCache({ cache_enabled: false, cache_size: 1024 });

            const { dialog, input } = await openDialog('dns_cache_size');

            fireEvent.change(input, { target: { value: '0' } });
            saveDialog(dialog);

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({
                    cache_size: 0,
                    cache_enabled: false,
                }),
            );
            await waitForSave();
        });

        it.each([false, true])(
            'rejects a fractional cache size when cache_enabled is %s',
            async (enabled) => {
                await renderCache({ cache_enabled: enabled, cache_size: 1024 });

                const { dialog, input } = await openDialog('dns_cache_size');

                fireEvent.change(input, { target: { value: '0.5' } });
                saveDialog(dialog);

                expect(mocks.apiDnsConfig).not.toHaveBeenCalled();
                expect(
                    within(dialog).getByText(
                        copyInDom('form_value_value_from_error', {
                            min_value: UINT32_RANGE.MIN.toLocaleString(),
                            max_value: UINT32_RANGE.MAX.toLocaleString(),
                        }),
                    ),
                ).toBeDefined();
            },
        );

        it.each(['dns_override_min_ttl', 'dns_override_max_ttl'] as const)(
            'rejects a fractional %s value',
            async (title) => {
                await renderCache();

                const { dialog, input } = await openDialog(title);

                fireEvent.change(input, { target: { value: '0.5' } });
                saveDialog(dialog);

                expect(mocks.apiDnsConfig).not.toHaveBeenCalled();
                expect(
                    within(dialog).getByText(
                        copyInDom('form_value_value_from_error', {
                            min_value: UINT32_RANGE.MIN.toLocaleString(),
                            max_value: UINT32_RANGE.MAX.toLocaleString(),
                        }),
                    ),
                ).toBeDefined();
            },
        );

        it('rejects a cache size above the uint32 range', async () => {
            await renderCache({ cache_size: 1024 });

            const { dialog, input } = await openDialog('dns_cache_size');

            fireEvent.change(input, { target: { value: '4294967296' } });
            saveDialog(dialog);

            expect(mocks.apiDnsConfig).not.toHaveBeenCalled();
            expect(
                within(dialog).getByText(
                    copyInDom('form_value_value_from_error', {
                        min_value: UINT32_RANGE.MIN.toLocaleString(),
                        max_value: UINT32_RANGE.MAX.toLocaleString(),
                    }),
                ),
            ).toBeDefined();
        });

        it('rejects a minimum TTL above the maximum', async () => {
            await renderCache({ cache_ttl_min: 0, cache_ttl_max: 100 });

            const { dialog, input } = await openDialog('dns_override_min_ttl');

            fireEvent.change(input, { target: { value: '200' } });
            saveDialog(dialog);

            expect(mocks.apiDnsConfig).not.toHaveBeenCalled();
            expect(
                within(dialog).getByText(copyInDom('cache_config_ttl_validation')),
            ).toBeDefined();
        });

        it('rejects a maximum TTL below the minimum', async () => {
            await renderCache({ cache_ttl_min: 100, cache_ttl_max: 200 });

            const { dialog, input } = await openDialog('dns_override_max_ttl');

            fireEvent.change(input, { target: { value: '50' } });
            saveDialog(dialog);

            expect(mocks.apiDnsConfig).not.toHaveBeenCalled();
            expect(
                within(dialog).getByText(copyInDom('cache_config_ttl_validation')),
            ).toBeDefined();
        });
    });

    describe('switches', () => {
        it('enables the cache', async () => {
            await renderCache({ cache_enabled: false });

            fireEvent.click(getSwitch('cache_enabled'));

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({ cache_enabled: true }),
            );
            await waitForSave();
        });

        it('disables the cache', async () => {
            await renderCache({ cache_enabled: true });

            fireEvent.click(getSwitch('cache_enabled'));

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({ cache_enabled: false }),
            );
            await waitForSave();
        });

        it('enables optimistic caching', async () => {
            await renderCache({ cache_enabled: true, cache_optimistic: false });

            fireEvent.click(getSwitch('optimistic_caching'));

            await waitFor(() =>
                expect(mocks.apiDnsConfig).toHaveBeenCalledWith({ cache_optimistic: true }),
            );
            await waitForSave();
        });

        it('reverts the cache switch when saving fails', async () => {
            mocks.apiDnsConfig.mockRejectedValueOnce(new Error('failed'));
            await renderCache({ cache_enabled: false });

            const input = getSwitch('cache_enabled');
            fireEvent.click(input);

            expect(input.checked).toBe(true);
            await waitFor(() => expect(input.checked).toBe(false));
        });

        it('ignores a second toggle while a save is in flight', async () => {
            const pending = deferred<void>();
            mocks.apiDnsConfig.mockReturnValueOnce(pending.promise);
            await renderCache({ cache_enabled: false });

            const input = getSwitch('cache_enabled');
            fireEvent.click(input);
            fireEvent.click(input);

            expect(mocks.apiDnsConfig).toHaveBeenCalledTimes(1);

            pending.resolve();
            await waitFor(() => expect(dnsConfigState.processingSetConfig).toBe(false));
        });
    });

    describe('clear cache', () => {
        it('clears the cache after confirmation', async () => {
            await renderCache();

            fireEvent.click(screen.getByText(copyInDom('dns_clear_cache')));
            fireEvent.click(screen.getByText(copyInDom('dns_clear_cache_confirm')));

            expect(mocks.apiCacheClear).toHaveBeenCalledTimes(1);
        });

        it('does not clear the cache when cancelled', async () => {
            await renderCache();

            fireEvent.click(screen.getByText(copyInDom('dns_clear_cache')));
            fireEvent.click(screen.getByText(copyInDom('cancel')));

            expect(mocks.apiCacheClear).not.toHaveBeenCalled();
        });
    });
});

describe('cache summaries', () => {
    it.each([
        [0, '0 bytes'],
        [512, '512 bytes'],
        [2048, '2 KB'],
        [1048576, '1 MB'],
        [4194304, '4 MB'],
    ])('formats %i bytes as %s', (bytes, expected) => {
        expect(getCacheSizeSummary(bytes)).toBe(expected);
    });

    it('returns an empty cache size summary for missing values', () => {
        expect(getCacheSizeSummary(undefined)).toBe('');
    });

    it('formats the TTL summary', () => {
        expect(getTtlSummary(60)).toBe(copy('dns_ttl_value', { value: 60 }));
    });

    it('returns an empty TTL summary for missing values', () => {
        expect(getTtlSummary(undefined)).toBe('');
    });
});
