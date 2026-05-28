import React, { useEffect, useState, useMemo } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { RootState } from 'panel/initialState';
import { Switch } from 'panel/common/controls/Switch';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Select } from 'panel/common/controls/Select';
import { Icon } from 'panel/common/ui/Icon';
import { Loader } from 'panel/common/ui/Loader';
import { toggleProtection, getClients } from 'panel/actions';
import { getStats, getStatsConfig } from 'panel/actions/stats';
import { getAccessList } from 'panel/actions/access';
import {
    DISABLE_PROTECTION_TIMINGS,
    ONE_SECOND_IN_MS,
    HOUR,
    DAY,
    STATS_INTERVALS_DAYS,
} from 'panel/helpers/constants';
import { Link } from 'react-router-dom';
import { msToSeconds, msToMinutes, msToHours } from 'panel/helpers/helpers';
import { Paths } from 'panel/components/Routes/Paths';

import { StatCards } from './blocks/StatCards';
import { GeneralStatistics } from './blocks/GeneralStatistics';
import { TopClients } from './blocks/TopClients';
import { TopQueriedDomains } from './blocks/TopQueriedDomains';
import { TopBlockedDomains } from './blocks/TopBlockedDomains';
import { TopUpstreams } from './blocks/TopUpstreams';
import { UpstreamAvgTime } from './blocks/UpstreamAvgTime';

import s from './Dashboard.module.pcss';

const DISABLE_PROTECTION_ITEMS = [
    {
        key: 'half_minute',
        time: DISABLE_PROTECTION_TIMINGS.HALF_MINUTE,
    },
    {
        key: 'minute',
        time: DISABLE_PROTECTION_TIMINGS.MINUTE,
    },
    {
        key: 'ten_minutes',
        time: DISABLE_PROTECTION_TIMINGS.TEN_MINUTES,
    },
    {
        key: 'hour',
        time: DISABLE_PROTECTION_TIMINGS.HOUR,
    },
    {
        key: 'tomorrow',
        time: DISABLE_PROTECTION_TIMINGS.TOMORROW,
    },
];

const getPeriodLabel = (interval: number) => {
    const hours = interval / (60 * 60 * 1000);
    if (hours === 1) {
        return intl.getMessage('last_hour');
    }
    if (hours === 24) {
        return intl.getPlural('last_hours', 24);
    }
    const days = hours / 24;
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
};

export const Dashboard = () => {
    const dispatch = useDispatch();
    const { dashboard, stats, access } = useSelector((state: RootState) => state);
    const [protectionMenuOpen, setProtectionMenuOpen] = useState(false);
    const [remainingTime, setRemainingTime] = useState<number | null>(null);
    const [selectedDisableTime, setSelectedDisableTime] = useState<number | null>(null);
    const [selectedPeriod, setSelectedPeriod] = useState(DAY);
    const [periodMenuOpen, setPeriodMenuOpen] = useState(false);
    const timerRef = React.useRef<ReturnType<typeof setInterval> | null>(null);

    const protectionDisabledDuration = dashboard?.protectionDisabledDuration;

    const startCountdown = React.useCallback((duration: number) => {
        if (timerRef.current) {
            clearInterval(timerRef.current);
        }
        setRemainingTime(duration);
        timerRef.current = setInterval(() => {
            setRemainingTime((prev) => {
                if (prev !== null && prev > ONE_SECOND_IN_MS) {
                    return prev - ONE_SECOND_IN_MS;
                }
                if (timerRef.current) {
                    clearInterval(timerRef.current);
                    timerRef.current = null;
                }
                dispatch(toggleProtection(false));
                return null;
            });
        }, ONE_SECOND_IN_MS);
    }, [dispatch]);

    useEffect(() => {
        if (protectionDisabledDuration && protectionDisabledDuration > 0 && timerRef.current === null) {
            startCountdown(protectionDisabledDuration);
        }
    }, [protectionDisabledDuration, startCountdown]);

    useEffect(() => () => {
        if (timerRef.current) {
            clearInterval(timerRef.current);
        }
    }, []);

    const maxStatsInterval = stats?.interval || DAY;
    const effectiveMaxStatsInterval = maxStatsInterval >= HOUR ? maxStatsInterval : DAY;
    const periodIntervals = useMemo(() => {
        const intervals = STATS_INTERVALS_DAYS.filter((interval) => interval <= effectiveMaxStatsInterval);

        if (!intervals.includes(effectiveMaxStatsInterval)) {
            intervals.push(effectiveMaxStatsInterval);
        }

        return intervals.sort((a, b) => a - b);
    }, [effectiveMaxStatsInterval]);

    const periodOptions = useMemo(
        () => periodIntervals.map((interval) => ({ value: interval, label: getPeriodLabel(interval) })),
        [periodIntervals],
    );

    useEffect(() => {
        const maxAvailable = periodIntervals[periodIntervals.length - 1];
        if (maxAvailable && selectedPeriod > maxAvailable) {
            setSelectedPeriod(maxAvailable);
        }
    }, [periodIntervals, selectedPeriod]);

    useEffect(() => {
        dispatch(getStats(selectedPeriod));
        dispatch(getStatsConfig());
        dispatch(getClients());
        dispatch(getAccessList());
    }, [dispatch, selectedPeriod]);

    if (!dashboard || !stats) {
        return (
            <div className={theme.layout.container}>
                <div className={theme.layout.containerIn}>
                    <div className={s.loader}>
                        <Loader />
                    </div>
                </div>
            </div>
        );
    }

    const {
        protectionEnabled,
        processingProtection,
    } = dashboard;

    const {
        processingStats,
        processingGetConfig,
        numDnsQueries,
        numBlockedFiltering,
        numReplacedSafebrowsing,
        numReplacedParental,
        numReplacedSafesearch,
        avgProcessingTime,
        dnsQueries,
        blockedFiltering,
        replacedSafebrowsing,
        replacedParental,
        topQueriedDomains,
        topBlockedDomains,
        topClients,
        topUpstreamsResponses,
        topUpstreamsAvgTime,
    } = stats;

    const handleRefreshStats = () => {
        dispatch(getStats(selectedPeriod));
        dispatch(getStatsConfig());
        dispatch(getClients());
        dispatch(getAccessList());
    };

    const handlePeriodChange = (period: number) => {
        setSelectedPeriod(period);
        setPeriodMenuOpen(false);
    };

    const handleToggleProtection = () => {
        if (!protectionEnabled && timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
            setRemainingTime(null);
            setSelectedDisableTime(null);
        }
        dispatch(toggleProtection(protectionEnabled));
    };

    const handleDisableProtection = (time: number) => {
        const duration = time - ONE_SECOND_IN_MS;
        setSelectedDisableTime(time);
        startCountdown(duration);
        dispatch(toggleProtection(protectionEnabled, duration));
        setProtectionMenuOpen(false);
    };

    const getDisableText = (key: string, time: number) => {
        switch (key) {
            case 'half_minute':
                return intl.translator.getPlural('pause_for_seconds', msToSeconds(time));
            case 'minute':
            case 'ten_minutes':
                return intl.translator.getPlural('pause_for_minutes', msToMinutes(time));
            case 'hour':
                return intl.getMessage('pause_for_hour', { count: msToHours(time) });
            case 'tomorrow': {
                const now = new Date();
                const tomorrowTime = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
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

    const isLoading = processingStats || processingGetConfig || access?.processing;

    const protectionMenu = (
        <div className={s.protectionMenu}>
            {DISABLE_PROTECTION_ITEMS.map((item) => (
                <div
                    key={item.key}
                    className={cn(theme.text.t2, theme.text.condenced, s.protectionMenuItem)}
                    onMouseDown={() => handleDisableProtection(item.time)}
                >
                    {selectedDisableTime === item.time && remainingTime ? (
                        <Icon icon="check_tiny" className={s.periodMenuIcon} />
                    ) : (
                        <span className={s.periodMenuDot}></span>
                    )}
                    {getDisableText(item.key, item.time)}
                </div>
            ))}
        </div>
    );

    return (
        <div className={theme.layout.container}>
            <div className={theme.layout.containerIn}>
                <div className={s.header}>
                    <div className={s.headerLeft}>
                        <h1 className={cn(theme.title.h5, s.onlyMobile)}>
                            {intl.getMessage('dashboard')}
                        </h1>

                        <h1 className={cn(theme.title.h3_tablet, s.onlyDesktop)}>
                            {intl.getMessage('protection')}
                        </h1>

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
                            wrapClassName={cn(s.protectionMenuWrapper, s.onlyDesktop)}
                            noIcon
                        >
                            <Icon icon="bullets" />
                        </Dropdown>

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
                            className={s.refreshButton}
                            onClick={handleRefreshStats}
                            disabled={isLoading}
                        >
                            <div className={s.onlyDesktop}>
                                {intl.getMessage('refresh_statics')}
                            </div>

                            <Icon icon="refresh" color="green" />
                        </button>

                        {protectionEnabled && (
                            <Dropdown
                                menu={protectionMenu}
                                trigger="click"
                                position="bottomLeft"
                                open={protectionMenuOpen}
                                onOpenChange={setProtectionMenuOpen}
                                wrapClassName={cn(s.protectionMenuWrapper, s.onlyMobile)}
                                noIcon
                            >
                                <Icon icon="bullets" />
                            </Dropdown>
                        )}

                        <Dropdown
                            wrapClassName={s.onlyDesktop}
                            menu={
                                <div className={s.periodMenu}>
                                    {periodOptions.map((option) => (
                                        <div
                                            key={option.value}
                                            className={cn(
                                                theme.text.t2,
                                                theme.text.condenced,
                                                s.periodMenuItem,
                                            )}
                                            onClick={() => handlePeriodChange(option.value)}
                                        >
                                            {selectedPeriod === option.value ? (
                                                <Icon icon="check_tiny" className={s.periodMenuIcon} />
                                            ) : (
                                                <span className={s.periodMenuDot}></span>
                                            )}
                                            {option.label}
                                        </div>
                                    ))}

                                    <div
                                        className={cn(s.periodMenuItem, s.periodMenuItemLink)}
                                    >
                                        <Icon icon="settings" className={s.periodMenuIcon} />

                                        <div className={cn(theme.text.t2, theme.text.condenced)}>
                                            {intl.getMessage('period_notify', {
                                                a: (text: string) => (
                                                    <Link
                                                        key="a"
                                                        to={{ pathname: Paths.SettingsPage, hash: '#stats_config' }}
                                                        className={cn(theme.link.link, theme.link.noDecoration)}
                                                    >
                                                        {text}
                                                    </Link>
                                                ),
                                            })}
                                        </div>
                                    </div>
                                </div>
                            }
                            trigger="click"
                            position="bottomRight"
                            open={periodMenuOpen}
                            onOpenChange={setPeriodMenuOpen}
                            noIcon
                        >
                            <button type="button" className={s.periodButton}>
                                <div className={cn(theme.text.t2, theme.text.condenced)}>
                                    {getPeriodLabel(selectedPeriod)}
                                </div>

                                <Icon icon="arrow_bottom" className={cn(s.periodButtonIcon, periodMenuOpen && s.periodButtonIconOpen)} />
                            </button>
                        </Dropdown>
                    </div>

                    <div className={cn(s.periodSelect, s.onlyMobile)}>
                        <Select<number>
                            options={periodOptions}
                            value={periodOptions.find((o) => o.value === selectedPeriod)}
                            onChange={(option) => handlePeriodChange(option.value)}
                            size="responsive"
                            height="big"
                            isSearchable={false}
                            components={{
                                MenuList: ({ children }: { children: React.ReactNode }) => (
                                    <div>
                                        {children}
                                        <div className={cn(s.periodMenuItem, s.periodMenuItemLink)}>
                                            <Icon icon="settings" className={s.periodMenuIcon} />
                                            <div className={cn(theme.text.t2, theme.text.condenced)}>
                                                {intl.getMessage('period_notify', {
                                                    a: (text: string) => (
                                                        <Link
                                                            key="a"
                                                            to={{ pathname: Paths.SettingsPage, hash: '#stats_config' }}
                                                            className={cn(theme.link.link, theme.link.noDecoration)}
                                                        >
                                                            {text}
                                                        </Link>
                                                    ),
                                                })}
                                            </div>
                                        </div>
                                    </div>
                                ),
                            }}
                        />
                    </div>
                </div>

                {isLoading ? (
                    <div className={s.loader}>
                        <Loader />
                    </div>
                ) : (
                    <>
                        <StatCards
                            numDnsQueries={numDnsQueries}
                            numBlockedFiltering={numBlockedFiltering}
                            numReplacedSafebrowsing={numReplacedSafebrowsing}
                            numReplacedParental={numReplacedParental}
                            dnsQueries={dnsQueries}
                            blockedFiltering={blockedFiltering}
                            replacedSafebrowsing={replacedSafebrowsing}
                            replacedParental={replacedParental}
                        />

                        <div className={s.statContainer}>
                            <GeneralStatistics
                                numDnsQueries={numDnsQueries}
                                numBlockedFiltering={numBlockedFiltering}
                                numReplacedSafebrowsing={numReplacedSafebrowsing}
                                numReplacedParental={numReplacedParental}
                                numReplacedSafesearch={numReplacedSafesearch}
                                avgProcessingTime={avgProcessingTime}
                            />

                            <TopClients
                                topClients={topClients}
                                numDnsQueries={numDnsQueries}
                            />

                            <TopQueriedDomains
                                topQueriedDomains={topQueriedDomains}
                                numDnsQueries={numDnsQueries}
                            />

                            <TopBlockedDomains
                                topBlockedDomains={topBlockedDomains}
                                numBlockedFiltering={numBlockedFiltering}
                            />

                            <TopUpstreams
                                topUpstreamsResponses={topUpstreamsResponses}
                                numDnsQueries={numDnsQueries}
                            />

                            <UpstreamAvgTime
                                topUpstreamsAvgTime={topUpstreamsAvgTime}
                                avgProcessingTime={avgProcessingTime}
                            />
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};
