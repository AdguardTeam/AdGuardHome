import { createSignal, Show, For } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Switch } from 'panel/common/controls/Switch';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Select } from 'panel/common/controls/Select';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { RoutePath, SCROLL_QUERY_KEY } from 'panel/components/Routes/Paths';
import { useIsMobile } from 'panel/hooks/useIsMobile';
import { DISABLE_PROTECTION_TIMINGS, ONE_SECOND_IN_MS } from 'panel/helpers/constants';
import { msToSeconds, msToMinutes, msToHours } from 'panel/helpers/helpers';

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
        if (days === 7) return intl.getPlural('last_days', 7);
        if (days === 30) return intl.getPlural('last_days', 30);
        if (days === 90) return intl.getPlural('last_days', 90);
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
    if (!milliseconds) return '';

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

export const Header = (props: Props) => {
    const [protectionMenuOpen, setProtectionMenuOpen] = createSignal(false);
    const [selectedDisableTime, setSelectedDisableTime] = createSignal<number | null>(null);
    const isMobile = useIsMobile();

    const handleToggleProtection = () => {
        props.onToggleProtection(props.protectionEnabled);
    };

    const handleDisableProtection = (time: number) => {
        const duration = time - ONE_SECOND_IN_MS;
        setSelectedDisableTime(time);
        props.onToggleProtection(props.protectionEnabled, duration);
        setProtectionMenuOpen(false);
    };

    const protectionMenu = (
        <div class={s.protectionMenu}>
            <For each={DISABLE_PROTECTION_ITEMS}>
                {(item) => (
                    <div
                        class={cn(
                            theme.select.option,
                            theme.select.option_check,
                            theme.text.t2,
                            theme.text.condenced,
                        )}
                        onMouseDown={() => handleDisableProtection(item.time)}
                    >
                        <Show
                            when={selectedDisableTime() === item.time && props.remainingTime}
                            fallback={<Icon icon="dot" class={theme.select.icon} />}
                        >
                            <Icon icon="check_tiny" class={theme.select.icon} />
                        </Show>
                        {getDisableText(item.key, item.time)}
                    </div>
                )}
            </For>
        </div>
    );

    const periodSettingsFooter = (
        <div class={cn(s.periodSettingsFooter, theme.select.option_check)}>
            <Icon icon="settings" class={theme.select.icon} />
            <div class={cn(theme.text.t2, theme.text.condenced)}>
                {intl.getMessage('period_notify', {
                    a: (text: string) => (
                        <Link
                            to={RoutePath.SettingsPage}
                            query={{ [SCROLL_QUERY_KEY]: 'statistics' }}
                            class={cn(theme.link.link, theme.link.noDecoration)}
                        >
                            {text}
                        </Link>
                    ),
                })}
            </div>
        </div>
    );

    return (
        <div class={s.header}>
            <div class={s.headerLeft}>
                <div class={s.titleRow}>
                    <h1 class={cn(theme.title.h5, s.onlyMobile)}>{intl.getMessage('dashboard')}</h1>

                    <button
                        type="button"
                        class={cn(s.refreshButton, s.refreshMobileButton, s.onlyMobile)}
                        onClick={() => props.onRefreshStats?.()}
                        disabled={props.isLoading}
                        aria-label={intl.getMessage('refresh_btn')}
                        title={intl.getMessage('refresh_btn')}
                    >
                        <Icon icon="refresh" color="green" />
                    </button>
                </div>

                <h1 class={cn(theme.title.h3_tablet, s.onlyDesktop)}>
                    {intl.getMessage('protection')}
                </h1>

                <div class={s.toggleRow}>
                    <div class={s.protectionToggle}>
                        <Switch
                            id="protection_toggle"
                            data-testid="protection-toggle"
                            checked={!!props.protectionEnabled}
                            onChange={handleToggleProtection}
                            disabled={props.processingProtection}
                        />

                        <div class={cn(theme.text.t2, s.onlyMobile)}>
                            {intl.getMessage('protection')}
                        </div>
                    </div>

                    <Dropdown
                        menu={protectionMenu}
                        trigger="click"
                        position="bottomLeft"
                        open={protectionMenuOpen()}
                        onOpenChange={setProtectionMenuOpen}
                        wrapClass={s.protectionMenuWrapper}
                        disabled={!props.protectionEnabled}
                        noIcon
                        disableAnimation
                    >
                        <button
                            type="button"
                            class={s.dropdownTrigger}
                            aria-label={intl.getMessage('disable_protection_btn')}
                            disabled={!props.protectionEnabled}
                        >
                            <Icon icon="bullets" />
                        </button>
                    </Dropdown>
                </div>

                <Show when={props.remainingTime && props.remainingTime > 0}>
                    <span class={s.cardSubtitle}>
                        {intl.getMessage('resume_protection_timer', {
                            time: getRemainingTimeText(props.remainingTime!),
                        })}
                    </span>
                </Show>
            </div>

            <div class={s.headerRight}>
                <button
                    type="button"
                    class={cn(s.refreshButton, s.refreshDesktopButton, s.onlyDesktop)}
                    onClick={() => props.onRefreshStats?.()}
                    disabled={props.isLoading}
                    aria-label={intl.getMessage('refresh_btn')}
                    title={intl.getMessage('refresh_btn')}
                >
                    {intl.getMessage('refresh_statics')}
                    <Icon icon="refresh" color="green" />
                </button>
            </div>

            <div class={s.periodSelect}>
                <Select<number>
                    options={props.periodOptions}
                    value={props.periodOptions.find((o) => o.value === props.selectedPeriod)}
                    onChange={(option: any) => props.onPeriodChange(option.value)}
                    size="responsive"
                    height="big"
                    isSearchable={false}
                    borderless={!isMobile()}
                    menuSize="big"
                    menuPosition="right"
                    menuFooter={periodSettingsFooter}
                />
            </div>
        </div>
    );
};
