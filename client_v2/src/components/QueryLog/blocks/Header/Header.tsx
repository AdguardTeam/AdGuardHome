import React, { useState, useCallback, useRef, useEffect } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import { Select } from 'panel/common/controls/Select';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { IOption } from 'panel/lib/helpers/utils';
import { RESPONSE_FILTER, DEBOUNCE_FILTER_TIMEOUT } from 'panel/helpers/constants';
import { useIsMobile } from 'panel/hooks/useIsMobile';

import s from './Header.module.pcss';

type Props = {
    onSearch: (value: string) => void;
    onRefresh: () => void;
    onFilterChange: (status: string) => void;
    currentSearch: string;
    currentFilter: string;
    isLoading: boolean;
};

const FILTER_OPTIONS = Object.values(RESPONSE_FILTER).map((filter) => ({
    value: filter.QUERY,
    label: filter.LABEL,
}));

export const Header = ({
    onSearch,
    onRefresh,
    onFilterChange,
    currentSearch,
    currentFilter,
    isLoading,
}: Props) => {
    const [searchValue, setSearchValue] = useState(currentSearch);
    const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const isMobile = useIsMobile();

    useEffect(() => {
        setSearchValue(currentSearch);
    }, [currentSearch]);

    const handleSearchChange = useCallback(
        (e: React.ChangeEvent<HTMLInputElement>) => {
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

    useEffect(() => {
        return () => {
            if (debounceRef.current) {
                clearTimeout(debounceRef.current);
            }
        };
    }, []);

    const selectedFilter = FILTER_OPTIONS.find((opt) => opt.value === currentFilter) || FILTER_OPTIONS[0];

    const handleFilterChange = useCallback(
        (option: IOption<string>) => {
            onFilterChange(option.value);
        },
        [onFilterChange],
    );

    return (
        <div className={s.header}>
            <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet, s.title)}>
                {intl.getMessage('query_log')}
            </h1>

            <div className={s.actions}>
                <div className={s.searchWrapper}>
                    <Input
                        type="text"
                        className={s.searchField}
                        placeholder={intl.getMessage('domain_or_client')}
                        value={searchValue}
                        onChange={handleSearchChange}
                        size="small"
                        prefixIcon={<Icon icon="search" className={s.searchIcon} />}
                        suffixIcon={<FaqTooltip text={intl.getMessage('query_log_strict_search')} />}
                    />

                    <Button
                        className={s.refreshMobileButton}
                        variant="primary"
                        size="small"
                        onClick={onRefresh}
                        disabled={isLoading}
                    >
                        <Icon icon="refresh" className={s.refreshMobileIcon} />
                    </Button>
                </div>

                <Button
                    className={s.refreshDesktopButton}
                    variant="ghost"
                    size="small"
                    rightAddon={<Icon icon="refresh" className={s.refreshDesktopIcon} />}
                    onClick={onRefresh}
                    disabled={isLoading}
                >
                    {intl.getMessage('refresh_btn')}
                </Button>

                <div className={s.filterField}>
                    <Select
                        size="responsive"
                        options={FILTER_OPTIONS}
                        value={selectedFilter}
                        onChange={handleFilterChange}
                        menuSize="medium"
                        menuPosition="right"
                        borderless={!isMobile}
                        className={s.filterSelect}
                    />
                </div>
            </div>
        </div>
    );
};
