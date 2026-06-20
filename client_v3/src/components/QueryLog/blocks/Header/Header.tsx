import { createSignal, createMemo, createRenderEffect, onCleanup, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import { Select } from 'panel/common/controls/Select';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { InlineLoader } from 'panel/common/ui/Loader';
import { IOption } from 'panel/lib/helpers/utils';
import {
    QUERY_LOG_REASON_FILTER,
    QUERY_LOG_STATUS_FILTER,
    DEBOUNCE_FILTER_TIMEOUT,
} from 'panel/helpers/constants';
import { useIsMobile } from 'panel/hooks/useIsMobile';

import s from './Header.module.pcss';

type Props = {
    onSearch: (value: string) => void;
    onRefresh: () => void;
    onStatusFilterChange: (status: string) => void;
    onReasonFilterChange: (reason: string) => void;
    currentSearch: string;
    currentStatus: string;
    currentReason: string;
    isLoading: boolean;
};

const STATUS_LABEL_KEYS: Record<string, string> = {
    all: 'query_log_all_statuses',
    processed: 'query_log_processed',
    allowed: 'query_log_allowed',
    blocked: 'query_log_blocked',
    rewritten: 'query_log_rewritten',
};

const REASON_LABEL_KEYS: Record<string, string> = {
    all: 'query_log_all_reasons',
    FilteredBlackList: 'query_log_blocked_by_filter',
    FilteredBlockedService: 'query_log_blocked_services',
    FilteredSafeBrowsing: 'query_log_blocked_threats',
    FilteredParental: 'query_log_blocked_by_parental_control',
    Rewrite: 'dns_rewrites',
    FilteredSafeSearch: 'query_log_safe_search',
};

export const Header = (props: Props) => {
    const [searchValue, setSearchValue] = createSignal('');
    let debounceTimer: ReturnType<typeof setTimeout> | null = null;
    const isMobile = useIsMobile();

    const statusOptions = createMemo(() =>
        Object.values(QUERY_LOG_STATUS_FILTER).map((filter) => ({
            value: filter.QUERY,
            label: intl.getMessage(STATUS_LABEL_KEYS[filter.QUERY]),
        })),
    );

    const reasonOptions = createMemo(() =>
        Object.values(QUERY_LOG_REASON_FILTER).map((filter) => ({
            value: filter.QUERY,
            label: intl.getMessage(REASON_LABEL_KEYS[filter.QUERY]),
        })),
    );

    createRenderEffect(() => {
        setSearchValue(props.currentSearch);
    });

    const handleSearchChange = (e: Event) => {
        const value = (e.target as HTMLInputElement).value;
        setSearchValue(value);

        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }

        debounceTimer = setTimeout(() => {
            props.onSearch(value);
        }, DEBOUNCE_FILTER_TIMEOUT);
    };

    const handleClearSearch = () => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }
        setSearchValue('');
        props.onSearch('');
    };

    onCleanup(() => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }
    });

    const selectedStatus = () =>
        statusOptions().find((opt) => opt.value === props.currentStatus) || statusOptions()[0];
    const selectedReason = () =>
        reasonOptions().find((opt) => opt.value === props.currentReason) || reasonOptions()[0];

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
                        onChange={handleSearchChange}
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
                        data-testid="query-log-refresh-button-mobile"
                        class={s.refreshMobileButton}
                        variant="primary"
                        size="small"
                        onClick={props.onRefresh}
                        disabled={props.isLoading}
                    >
                        <Icon icon="refresh" class={s.refreshMobileIcon} />
                    </Button>
                </div>

                <div class={s.filters}>
                    <div class={s.filterField} data-testid="query-log-status-filter">
                        <Select
                            size="responsive"
                            options={statusOptions()}
                            value={selectedStatus()}
                            optionTestIdPrefix="query-log-status-option"
                            onChange={(option: IOption<string>) =>
                                props.onStatusFilterChange(option.value)
                            }
                            menuSize="big"
                            menuPosition="right"
                            borderless={!isMobile()}
                            class={s.filterSelect}
                        />
                    </div>

                    <div class={s.filterField} data-testid="query-log-reason-filter">
                        <Select
                            size="responsive"
                            options={reasonOptions()}
                            value={selectedReason()}
                            optionTestIdPrefix="query-log-reason-option"
                            onChange={(option: IOption<string>) =>
                                props.onReasonFilterChange(option.value)
                            }
                            menuSize="big"
                            menuPosition="right"
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
                    disabled={props.isLoading}
                >
                    <Icon icon="refresh" class={s.refreshDesktopIcon} />
                </Button>
            </div>
        </div>
    );
};
