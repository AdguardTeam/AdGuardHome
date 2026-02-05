import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { RootState } from 'panel/initialState';
import { Switch } from 'panel/common/controls/Switch';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import { Loader } from 'panel/common/ui/Loader';
import { toggleProtection, getClients } from 'panel/actions';
import { getStats, getStatsConfig } from 'panel/actions/stats';
import { getAccessList } from 'panel/actions/access';
import {
    DISABLE_PROTECTION_TIMINGS,
    ONE_SECOND_IN_MS,
    DAY,
    STATS_INTERVALS_DAYS,
} from 'panel/helpers/constants';
import { msToSeconds, msToMinutes, msToHours } from 'panel/helpers/helpers';

import { StatCards } from './blocks/StatCards';
import { GeneralStatistics } from './blocks/GeneralStatistics';
import { TopClients } from './blocks/TopClients';
import { TopQueriedDomains } from './blocks/TopQueriedDomains';
import { TopBlockedDomains } from './blocks/TopBlockedDomains';
import { TopUpstreams } from './blocks/TopUpstreams';
import { UpstreamAvgTime } from './blocks/UpstreamAvgTime';
import { MOCK_DATA, USE_MOCK_DATA } from './mockData';

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
    return intl.getMessage('interval_days', { count: days });
};

const PERIOD_OPTIONS = STATS_INTERVALS_DAYS.map((interval) => ({
    value: interval,
    label: getPeriodLabel(interval),
}));

export const Dashboard = () => {
    const dispatch = useDispatch();
    const { dashboard, stats, access } = useSelector((state: RootState) => state);
    const [protectionMenuOpen, setProtectionMenuOpen] = useState(false);
    const [remainingTime, setRemainingTime] = useState<number | null>(null);
    const [selectedPeriod, setSelectedPeriod] = useState(DAY * 7);
    const [periodMenuOpen, setPeriodMenuOpen] = useState(false);

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
        protectionDisabledDuration,
    } = dashboard;

    useEffect(() => {
        if (protectionDisabledDuration && protectionDisabledDuration > 0) {
            setRemainingTime(protectionDisabledDuration);

            const timer = setInterval(() => {
                setRemainingTime((prev) => {
                    if (prev && prev > ONE_SECOND_IN_MS) {
                        return prev - ONE_SECOND_IN_MS;
                    }
                    clearInterval(timer);
                    dispatch(toggleProtection(false));
                    return null;
                });
            }, ONE_SECOND_IN_MS);

            return () => clearInterval(timer);
        }
        setRemainingTime(null);
        return undefined;
    }, [protectionDisabledDuration, dispatch]);

    const {
        processingStats,
        processingGetConfig,
        interval,
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
        timeUnits,
    } = stats;

    useEffect(() => {
        dispatch(getStats(selectedPeriod));
        dispatch(getStatsConfig());
        dispatch(getClients());
        dispatch(getAccessList());
    }, [dispatch, selectedPeriod]);

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
        dispatch(toggleProtection(protectionEnabled));
    };

    const handleDisableProtection = (time: number) => {
        dispatch(toggleProtection(protectionEnabled, time - ONE_SECOND_IN_MS));
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
                    className={s.protectionMenuItem}
                    onClick={() => handleDisableProtection(item.time)}
                >
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
                        <h1 className={cn(theme.title.h4, theme.title.h3_tablet)}>
                            {intl.getMessage('protection')}
                        </h1>

                        <Switch
                            id="protection_toggle"
                            data-testid="protection-toggle"
                            checked={!!protectionEnabled}
                            onChange={handleToggleProtection}
                            disabled={processingProtection}
                        />

                        {protectionEnabled && (
                            <Dropdown
                                menu={protectionMenu}
                                trigger="click"
                                position="bottomLeft"
                                open={protectionMenuOpen}
                                onOpenChange={setProtectionMenuOpen}
                                wrapClassName={s.protectionMenuWrapper}
                                noIcon
                            >
                                <Icon icon="bullets" />
                            </Dropdown>
                        )}

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
                            {intl.getMessage('refresh_statics')}

                            <Icon icon="refresh" color="green" />
                        </button>

                        <Dropdown
                            menu={
                                <div className={s.periodMenu}>
                                    {PERIOD_OPTIONS.map((option) => (
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
                </div>

                {isLoading ? (
                    <div className={s.loader}>
                        <Loader />
                    </div>
                ) : (
                    <>
                        <StatCards
                            numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                            numBlockedFiltering={
                                USE_MOCK_DATA ? MOCK_DATA.numBlockedFiltering : (numBlockedFiltering || 0)
                            }
                            numReplacedSafebrowsing={
                                USE_MOCK_DATA ? MOCK_DATA.numReplacedSafebrowsing : (numReplacedSafebrowsing || 0)
                            }
                            numReplacedParental={
                                USE_MOCK_DATA ? MOCK_DATA.numReplacedParental : (numReplacedParental || 0)
                            }
                            dnsQueries={USE_MOCK_DATA ? MOCK_DATA.dnsQueries : (dnsQueries || [])}
                            blockedFiltering={USE_MOCK_DATA ? MOCK_DATA.blockedFiltering : (blockedFiltering || [])}
                            replacedSafebrowsing={
                                USE_MOCK_DATA ? MOCK_DATA.replacedSafebrowsing : (replacedSafebrowsing || [])
                            }
                            replacedParental={USE_MOCK_DATA ? MOCK_DATA.replacedParental : (replacedParental || [])}
                        />

                        <div className={s.statContainer}>
                            <GeneralStatistics
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                                numBlockedFiltering={
                                    USE_MOCK_DATA ? MOCK_DATA.numBlockedFiltering : (numBlockedFiltering || 0)
                                }
                                numReplacedSafebrowsing={
                                    USE_MOCK_DATA ? MOCK_DATA.numReplacedSafebrowsing : (numReplacedSafebrowsing || 0)
                                }
                                numReplacedParental={
                                    USE_MOCK_DATA ? MOCK_DATA.numReplacedParental : (numReplacedParental || 0)
                                }
                                numReplacedSafesearch={
                                    USE_MOCK_DATA ? MOCK_DATA.numReplacedSafesearch : (numReplacedSafesearch || 0)
                                }
                                avgProcessingTime={
                                    USE_MOCK_DATA ? MOCK_DATA.avgProcessingTime : (avgProcessingTime || 0)
                                }
                            />

                            <TopClients
                                topClients={USE_MOCK_DATA ? MOCK_DATA.topClients : (topClients || [])}
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                            />

                            <TopQueriedDomains
                                topQueriedDomains={
                                    USE_MOCK_DATA ? MOCK_DATA.topQueriedDomains : (topQueriedDomains || [])
                                }
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                            />

                            <TopBlockedDomains
                                topBlockedDomains={
                                    USE_MOCK_DATA ? MOCK_DATA.topBlockedDomains : (topBlockedDomains || [])
                                }
                                numBlockedFiltering={
                                    USE_MOCK_DATA ? MOCK_DATA.numBlockedFiltering : (numBlockedFiltering || 0)
                                }
                            />

                            <TopUpstreams
                                topUpstreamsResponses={
                                    USE_MOCK_DATA ? MOCK_DATA.topUpstreamsResponses : (topUpstreamsResponses || [])
                                }
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                            />

                            <UpstreamAvgTime
                                topUpstreamsAvgTime={
                                    USE_MOCK_DATA ? MOCK_DATA.topUpstreamsAvgTime : (topUpstreamsAvgTime || [])
                                }
                                avgProcessingTime={
                                    USE_MOCK_DATA ? MOCK_DATA.avgProcessingTime : (avgProcessingTime || 0)
                                }
                            />
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};
