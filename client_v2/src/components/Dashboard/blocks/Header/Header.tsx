import React, { useState, useCallback, ReactNode } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Switch } from 'panel/common/controls/Switch';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Select } from 'panel/common/controls/Select';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'react-router-dom';
import { useIsMobile } from 'panel/hooks/useIsMobile';
import { DISABLE_PROTECTION_TIMINGS, ONE_SECOND_IN_MS } from 'panel/helpers/constants';
import { msToSeconds, msToMinutes, msToHours } from 'panel/helpers/helpers';
import { Paths } from 'panel/components/Routes/Paths';

import s from './Header.module.pcss';

const DISABLE_PROTECTION_ITEMS = [
    { key: 'half_minute', time: DISABLE_PROTECTION_TIMINGS.HALF_MINUTE },
    { key: 'minute', time: DISABLE_PROTECTION_TIMINGS.MINUTE },
    { key: 'ten_minutes', time: DISABLE_PROTECTION_TIMINGS.TEN_MINUTES },
    { key: 'hour', time: DISABLE_PROTECTION_TIMINGS.HOUR },
    { key: 'tomorrow', time: DISABLE_PROTECTION_TIMINGS.TOMORROW },
];

export const getPeriodLabel = (interval: number) => {
    const hours = interval / (60 * 60 * 1000);
    if (hours === 24) {
        return intl.getPlural('last_hours', 24);
    }
    const days = hours / 24;
    if (Number.isInteger(days)) {
        if (days === 7) {
            return intl.getPlural('last_days', 7);
        }
        if (days === 30) {
            return intl.getPlural('last_days', 30);
        }
        if (days === 90) {
            return intl.getPlural('last_days', 90);
        }
        return intl.getPlural('last_days', days);
    }
    return intl.getPlural('last_hours', Math.floor(hours));
};

const getDisableText = (key: string, time: number) => {
    switch (key) {
        case 'half_minute':
            return intl.getPlural('pause_for_seconds', msToSeconds(time));
        case 'minute':
        case 'ten_minutes':
            return intl.getPlural('pause_for_minutes', msToMinutes(time));
        case 'hour':
            return intl.getMessage('pause_for_hour', { count: msToHours(time) });
        case 'tomorrow': {
            const now = new Date();
            const tomorrowTime = now.toLocaleTimeString([], {
                hour: '2-digit',
                minute: '2-digit',
            });
            return intl.getMessage('pause_until_tomorrow', { time: tomorrowTime });
        }
        default:
            return '';
    }
};

const getRemainingTimeText = (milliseconds: number) => {
    if (!milliseconds) {
        return '';
    }

    const date = new Date(milliseconds);
    const hh = date.getUTCHours();
    const mm = `0${date.getUTCMinutes()}`.slice(-2);
    const ss = `0${date.getUTCSeconds()}`.slice(-2);
    const formattedHH = `0${hh}`.slice(-2);

    return hh ? `${formattedHH}:${mm}:${ss}` : `${mm}:${ss}`;
};

type Props = {
    protectionEnabled: boolean;
    processingProtection: boolean;
    remainingTime: number | null;
    selectedPeriod: number;
    periodOptions: Array<{ value: number; label: string }>;
    isLoading: boolean;
    onToggleProtection: (enabled: boolean, duration?: number) => void;
    onRefreshStats: () => void;
    onPeriodChange: (period: number) => void;
};

export const Header = ({
    protectionEnabled,
    processingProtection,
    remainingTime,
    selectedPeriod,
    periodOptions,
    isLoading,
    onToggleProtection,
    onRefreshStats,
    onPeriodChange,
}: Props) => {
    const [protectionMenuOpen, setProtectionMenuOpen] = useState(false);
    const [selectedDisableTime, setSelectedDisableTime] = useState<number | null>(null);
    const isMobile = useIsMobile();

    const handleToggleProtection = () => {
        onToggleProtection(protectionEnabled);
    };

    const handleDisableProtection = (time: number) => {
        const duration = time - ONE_SECOND_IN_MS;
        setSelectedDisableTime(time);
        onToggleProtection(protectionEnabled, duration);
        setProtectionMenuOpen(false);
    };

    const protectionMenu = (
        <div className={s.protectionMenu}>
            {DISABLE_PROTECTION_ITEMS.map((item) => (
                <div
                    key={item.key}
                    className={cn(
                        theme.select.option,
                        theme.select.option_check,
                        theme.text.t2,
                        theme.text.condenced,
                    )}
                    onMouseDown={() => handleDisableProtection(item.time)}
                >
                    {selectedDisableTime === item.time && remainingTime ? (
                        <Icon icon="check_tiny" className={theme.select.icon} />
                    ) : (
                        <Icon icon="dot" className={theme.select.icon} />
                    )}
                    {getDisableText(item.key, item.time)}
                </div>
            ))}
        </div>
    );

    const periodSettingsFooter = (
        <div className={cn(s.periodSettingsFooter, theme.select.option_check)}>
            <Icon icon="settings" className={theme.select.icon} />
            <div className={cn(theme.text.t2, theme.text.condenced)}>
                {intl.getMessage('period_notify', {
                    a: (text: string) => (
                        <Link
                            key="a"
                            to={{
                                pathname: Paths.SettingsPage,
                                hash: '#stats_config',
                            }}
                            className={cn(theme.link.link, theme.link.noDecoration)}
                        >
                            {text}
                        </Link>
                    ),
                })}
            </div>
        </div>
    );

    const periodSelectMenuList = useCallback(
        ({ children }: { children: ReactNode }) => (
            <div>
                {children}
                {periodSettingsFooter}
            </div>
        ),
        [],
    );

    return (
        <div className={s.header}>
            <div className={s.headerLeft}>
                <div className={s.titleRow}>
                    <h1 className={cn(theme.title.h5, s.onlyMobile)}>
                        {intl.getMessage('dashboard')}
                    </h1>

                    <button
                        type="button"
                        className={cn(s.refreshButton, s.refreshMobileButton, s.onlyMobile)}
                        onClick={onRefreshStats}
                        disabled={isLoading}
                        aria-label={intl.getMessage('refresh_btn')}
                        title={intl.getMessage('refresh_btn')}
                    >
                        <Icon icon="refresh" color="green" />
                    </button>
                </div>

                <h1 className={cn(theme.title.h3_tablet, s.onlyDesktop)}>
                    {intl.getMessage('protection')}
                </h1>

                <div className={s.toggleRow}>
                    <div className={s.protectionToggle}>
                        <Switch
                            id="protection_toggle"
                            data-testid="protection-toggle"
                            checked={!!protectionEnabled}
                            onChange={handleToggleProtection}
                            disabled={processingProtection}
                        />

                        <div className={cn(theme.text.t2, s.onlyMobile)}>
                            {intl.getMessage('protection')}
                        </div>
                    </div>

                    <Dropdown
                        menu={protectionMenu}
                        trigger="click"
                        position="bottomLeft"
                        open={protectionMenuOpen}
                        onOpenChange={setProtectionMenuOpen}
                        wrapClassName={s.protectionMenuWrapper}
                        disabled={!protectionEnabled}
                        noIcon
                        disableAnimation
                    >
                        <button
                            type="button"
                            className={s.dropdownTrigger}
                            aria-label={intl.getMessage('disable_protection_btn')}
                            disabled={!protectionEnabled}
                        >
                            <Icon icon="bullets" />
                        </button>
                    </Dropdown>
                </div>

                {remainingTime && remainingTime > 0 && (
                    <span className={s.cardSubtitle}>
                        {intl.getMessage('resume_protection_timer', {
                            time: getRemainingTimeText(remainingTime),
                        })}
                    </span>
                )}
            </div>

            <div className={s.headerRight}>
                <button
                    type="button"
                    className={cn(s.refreshButton, s.refreshDesktopButton, s.onlyDesktop)}
                    onClick={onRefreshStats}
                    disabled={isLoading}
                    aria-label={intl.getMessage('refresh_btn')}
                    title={intl.getMessage('refresh_btn')}
                >
                    {intl.getMessage('refresh_statics')}
                    <Icon icon="refresh" color="green" />
                </button>
            </div>

            <div className={s.periodSelect}>
                <Select<number>
                    options={periodOptions}
                    value={periodOptions.find((o) => o.value === selectedPeriod)}
                    onChange={(option) => onPeriodChange(option.value)}
                    size="responsive"
                    height="big"
                    isSearchable={false}
                    borderless={!isMobile}
                    menuSize="big"
                    menuPosition="right"
                    components={{
                        MenuList: periodSelectMenuList,
                    }}
                />
            </div>
        </div>
    );
};
