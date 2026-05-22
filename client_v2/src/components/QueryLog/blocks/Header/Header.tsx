import React, { useState, useCallback, useRef, useEffect } from 'react';
import type { ChangeEvent } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import { Select } from 'panel/common/controls/Select';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
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

const STATUS_OPTIONS = Object.values(QUERY_LOG_STATUS_FILTER).map((filter) => ({
    value: filter.QUERY,
    label: filter.LABEL,
}));

const REASON_OPTIONS = Object.values(QUERY_LOG_REASON_FILTER).map((filter) => ({
    value: filter.QUERY,
    label: filter.LABEL,
}));

export const Header = ({
    onSearch,
    onRefresh,
    onStatusFilterChange,
    onReasonFilterChange,
    currentSearch,
    currentStatus,
    currentReason,
    isLoading,
}: Props) => {
    const [searchValue, setSearchValue] = useState(currentSearch);
    const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const isMobile = useIsMobile();

    useEffect(() => {
        setSearchValue(currentSearch);
    }, [currentSearch]);

    const handleSearchChange = useCallback(
        (e: ChangeEvent<HTMLInputElement>) => {
            const { value } = e.target;
            setSearchValue(value);

            if (debounceRef.current) {
                clearTimeout(debounceRef.current);
            }

            debounceRef.current = setTimeout(() => {
                onSearch(value);
            }, DEBOUNCE_FILTER_TIMEOUT);
        },
        [onSearch],
    );

    const handleClearSearch = useCallback(() => {
        if (debounceRef.current) {
            clearTimeout(debounceRef.current);
        }

        setSearchValue('');
        onSearch('');
    }, [onSearch]);

    useEffect(() => {
        return () => {
            if (debounceRef.current) {
                clearTimeout(debounceRef.current);
            }
        };
    }, []);

    const selectedStatus = STATUS_OPTIONS.find((opt) => opt.value === currentStatus) || STATUS_OPTIONS[0];
    const selectedReason = REASON_OPTIONS.find((opt) => opt.value === currentReason) || REASON_OPTIONS[0];

    return (
        <div className={s.header}>
            <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet, s.title)}>
                {intl.getMessage('query_log')}
            </h1>

            <div className={s.actions}>
                <div className={s.searchWrapper}>
                    <Input
                        data-testid="query-log-search-input"
                        type="text"
                        className={s.searchField}
                        placeholder={intl.getMessage('domain_or_client')}
                        value={searchValue}
                        onChange={handleSearchChange}
                        size="small"
                        prefixIcon={<Icon icon="search" className={s.searchIcon} />}
                        suffixIcon={(
                            <div className={s.searchSuffix}>
                                {searchValue && (
                                    <button
                                        type="button"
                                        className={s.searchClearButton}
                                        data-testid="query-log-search-clear-button"
                                        aria-label={intl.getMessage('reset')}
                                        title={intl.getMessage('reset')}
                                        onMouseDown={(event) => event.preventDefault()}
                                        onClick={handleClearSearch}
                                    >
                                        <Icon icon="cross" className={s.searchClearIcon} />
                                    </button>
                                )}

                                <FaqTooltip text={intl.getMessage('query_log_strict_search')} />
                            </div>
                        )}
                    />

                    <Button
                        data-testid="query-log-refresh-button-mobile"
                        className={s.refreshMobileButton}
                        variant="primary"
                        size="small"
                        onClick={onRefresh}
                        disabled={isLoading}
                    >
                        <Icon icon="refresh" className={s.refreshMobileIcon} />
                    </Button>
                </div>

                <div className={s.filters}>
                    <div
                        className={s.filterField}
                        data-testid="query-log-status-filter"
                    >
                        <Select
                            size="responsive"
                            options={STATUS_OPTIONS}
                            value={selectedStatus}
                            optionTestIdPrefix="query-log-status-option"
                            onChange={(option: IOption<string>) => onStatusFilterChange(option.value)}
                            menuSize="medium"
                            menuPosition="right"
                            borderless={!isMobile}
                            className={s.filterSelect}
                        />
                    </div>

                    <div
                        className={s.filterField}
                        data-testid="query-log-reason-filter"
                    >
                        <Select
                            size="responsive"
                            options={REASON_OPTIONS}
                            value={selectedReason}
                            optionTestIdPrefix="query-log-reason-option"
                            onChange={(option: IOption<string>) => onReasonFilterChange(option.value)}
                            menuSize="medium"
                            menuPosition="right"
                            borderless={!isMobile}
                            className={s.filterSelect}
                        />
                    </div>
                </div>

                <Button
                    data-testid="query-log-refresh-button-desktop"
                    className={s.refreshDesktopButton}
                    variant="ghost"
                    size="small"
                    aria-label={intl.getMessage('refresh_btn')}
                    title={intl.getMessage('refresh_btn')}
                    onClick={onRefresh}
                    disabled={isLoading}
                >
                    <Icon icon="refresh" className={s.refreshDesktopIcon} />
                </Button>
            </div>
        </div>
    );
};
