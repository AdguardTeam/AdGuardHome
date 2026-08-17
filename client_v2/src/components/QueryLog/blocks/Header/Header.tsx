import { createSignal, createMemo, createEffect, on, onCleanup, Show, untrack } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Icon } from 'panel/common/ui/Icon';
import { Select } from 'panel/common/controls/Select';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { InlineLoader } from 'panel/common/ui/Loader';
import { IOption } from 'panel/lib/helpers/utils';
import { DEBOUNCE_FILTER_TIMEOUT } from 'panel/helpers/constants';
import { useIsMobile } from 'panel/hooks/useIsMobile';

import s from './Header.module.pcss';

type Props = {
    onSearch: (value: string) => void;
    onRefresh: () => void;
    onClear: () => Promise<void>;
    onStatusFilterChange: (status: string) => void;
    onReasonFilterChange: (reason: string) => void;
    currentSearch: string;
    currentStatus: string;
    currentReason: string;
    isLoading: boolean;
    isClearing: boolean;
};

const STATUS_OPTIONS = [
    {
        value: 'all',
        get label() {
            return intl.getMessage('query_log_all_statuses');
        },
    },
    {
        value: 'processed',
        get label() {
            return intl.getMessage('query_log_processed');
        },
    },
    {
        value: 'allowed',
        get label() {
            return intl.getMessage('query_log_allowed');
        },
    },
    {
        value: 'blocked',
        get label() {
            return intl.getMessage('query_log_blocked');
        },
    },
    {
        value: 'rewritten',
        get label() {
            return intl.getMessage('query_log_rewritten');
        },
    },
];

const REASON_OPTIONS = [
    {
        value: 'all',
        get label() {
            return intl.getMessage('query_log_all_reasons');
        },
    },
    {
        value: 'FilteredBlackList',
        get label() {
            return intl.getMessage('query_log_blocked_by_filter');
        },
    },
    {
        value: 'FilteredBlockedService',
        get label() {
            return intl.getMessage('query_log_blocked_services');
        },
    },
    {
        value: 'FilteredSafeBrowsing',
        get label() {
            return intl.getMessage('query_log_blocked_threats');
        },
    },
    {
        value: 'FilteredParental',
        get label() {
            return intl.getMessage('query_log_blocked_by_parental_control');
        },
    },
    {
        value: 'Rewrite',
        get label() {
            return intl.getMessage('dns_rewrites');
        },
    },
    {
        value: 'FilteredSafeSearch',
        get label() {
            return intl.getMessage('query_log_safe_search');
        },
    },
];

export const Header = (props: Props) => {
    const [searchValue, setSearchValue] = createSignal(untrack(() => props.currentSearch));
    const [showClearConfirm, setShowClearConfirm] = createSignal(false);
    let debounceTimer: ReturnType<typeof setTimeout> | null = null;
    const isMobile = useIsMobile();
    const isBusy = () => props.isLoading || props.isClearing;

    const cancelDebouncedSearch = () => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
            debounceTimer = null;
        }
    };

    createEffect(
        on(
            () => props.currentSearch,
            (value) => setSearchValue(value),
            { defer: true },
        ),
    );

    const handleSearchChange = (e: Event) => {
        const value = (e.target as HTMLInputElement).value;
        setSearchValue(value);

        cancelDebouncedSearch();

        debounceTimer = setTimeout(() => {
            props.onSearch(value);
        }, DEBOUNCE_FILTER_TIMEOUT);
    };

    const handleClearSearch = () => {
        cancelDebouncedSearch();
        setSearchValue('');
        props.onSearch('');
    };

    const handleOpenClearConfirm = () => {
        cancelDebouncedSearch();
        setShowClearConfirm(true);
    };

    const handleClear = async () => {
        await props.onClear();
        setShowClearConfirm(false);
    };

    onCleanup(() => {
        cancelDebouncedSearch();
    });

    const selectedStatus = createMemo(
        () => STATUS_OPTIONS.find((opt) => opt.value === props.currentStatus) || STATUS_OPTIONS[0],
    );
    const selectedReason = createMemo(
        () => REASON_OPTIONS.find((opt) => opt.value === props.currentReason) || REASON_OPTIONS[0],
    );

    return (
        <div class={s.header}>
            <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet, s.title)}>
                {intl.getMessage('query_log')}
            </h1>

            <div class={s.actions}>
                <div class={s.searchWrapper}>
                    <Input
                        data-testid="query-log-search-input"
                        type="text"
                        class={s.searchField}
                        placeholder={intl.getMessage('domain_or_client')}
                        value={searchValue()}
                        onInput={handleSearchChange}
                        disabled={isBusy()}
                        size="small"
                        prefixIcon={<Icon icon="search" class={s.searchIcon} />}
                        suffixIcon={
                            <div class={s.searchSuffix}>
                                <Show
                                    when={props.isLoading}
                                    fallback={
                                        <Show when={searchValue()}>
                                            <button
                                                type="button"
                                                class={s.searchClearButton}
                                                data-testid="query-log-search-clear-button"
                                                aria-label={intl.getMessage('reset')}
                                                title={intl.getMessage('reset')}
                                                disabled={isBusy()}
                                                onMouseDown={(event) => event.preventDefault()}
                                                onClick={handleClearSearch}
                                            >
                                                <Icon icon="cross" class={s.searchClearIcon} />
                                            </button>
                                        </Show>
                                    }
                                >
                                    <InlineLoader class={s.searchLoader} />
                                </Show>

                                <FaqTooltip text={intl.getMessage('query_log_strict_search')} />
                            </div>
                        }
                    />

                    <Button
                        data-testid="query-log-clear-button-mobile"
                        class={s.clearMobileButton}
                        variant="secondary-danger"
                        size="small"
                        aria-label={intl.getMessage('clear_query_log')}
                        title={intl.getMessage('clear_query_log')}
                        onClick={handleOpenClearConfirm}
                        disabled={isBusy()}
                    >
                        <Icon icon="delete" class={s.clearIcon} />
                    </Button>

                    <Button
                        data-testid="query-log-refresh-button-mobile"
                        class={s.refreshMobileButton}
                        variant="primary"
                        size="small"
                        onClick={props.onRefresh}
                        disabled={isBusy()}
                    >
                        <Icon icon="refresh" class={s.refreshMobileIcon} />
                    </Button>
                </div>

                <div class={s.filters}>
                    <div class={s.filterField} data-testid="query-log-status-filter">
                        <Select
                            size="responsive"
                            options={STATUS_OPTIONS}
                            value={selectedStatus()}
                            optionTestIdPrefix="query-log-status-option"
                            onChange={(option: IOption<string>) =>
                                props.onStatusFilterChange(option.value)
                            }
                            menuSize="big"
                            menuPosition="right"
                            isDisabled={isBusy()}
                            borderless={!isMobile()}
                            class={s.filterSelect}
                        />
                    </div>

                    <div class={s.filterField} data-testid="query-log-reason-filter">
                        <Select
                            size="responsive"
                            options={REASON_OPTIONS}
                            value={selectedReason()}
                            optionTestIdPrefix="query-log-reason-option"
                            onChange={(option: IOption<string>) =>
                                props.onReasonFilterChange(option.value)
                            }
                            menuSize="big"
                            menuPosition="right"
                            isDisabled={isBusy()}
                            borderless={!isMobile()}
                            class={s.filterSelect}
                        />
                    </div>
                </div>

                <Button
                    data-testid="query-log-refresh-button-desktop"
                    class={s.refreshDesktopButton}
                    variant="ghost"
                    size="small"
                    aria-label={intl.getMessage('refresh_btn')}
                    title={intl.getMessage('refresh_btn')}
                    onClick={props.onRefresh}
                    disabled={isBusy()}
                >
                    <Icon icon="refresh" class={s.refreshDesktopIcon} />
                </Button>

                <Button
                    data-testid="query-log-clear-button-desktop"
                    class={s.clearDesktopButton}
                    variant="secondary-danger"
                    size="small"
                    aria-label={intl.getMessage('clear_query_log')}
                    title={intl.getMessage('clear_query_log')}
                    onClick={handleOpenClearConfirm}
                    disabled={isBusy()}
                >
                    <Icon icon="delete" class={s.clearIcon} />
                </Button>
            </div>

            <Show when={showClearConfirm()}>
                <ConfirmDialog
                    title={intl.getMessage('settings_confirm_clear_query_log')}
                    text={intl.getMessage('settings_confirm_clear_query_log_desc')}
                    buttonText={intl.getMessage('settings_yes_clear')}
                    cancelText={intl.getMessage('cancel')}
                    buttonVariant="danger"
                    submitDisabled={isBusy()}
                    submitTestId="query-log-clear-confirm"
                    onClose={() => setShowClearConfirm(false)}
                    onConfirm={handleClear}
                />
            </Show>
        </div>
    );
};
