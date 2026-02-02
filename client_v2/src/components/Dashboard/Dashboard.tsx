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
import { msToSeconds, msToMinutes, msToHours, msToDays } from 'panel/helpers/helpers';

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

const PERIOD_OPTIONS = STATS_INTERVALS_DAYS.map((interval) => ({
    value: interval,
    label: intl.getMessage('interval_days', { count: msToDays(interval) }),
}));

// Mock data for testing charts
const MOCK_DATA = {
    dnsQueries: [120, 150, 180, 200, 170, 190, 220, 250, 230, 210, 240, 280, 300, 320, 290, 310, 350, 380, 360, 340, 370, 400, 420, 390],
    blockedFiltering: [10, 15, 12, 18, 14, 20, 25, 22, 28, 24, 30, 35, 32, 38, 34, 40, 45, 42, 48, 44, 50, 55, 52, 58],
    replacedSafebrowsing: [2, 3, 1, 4, 2, 5, 3, 6, 4, 7, 5, 8, 6, 9, 7, 10, 8, 11, 9, 12, 10, 13, 11, 14],
    replacedParental: [1, 2, 1, 3, 2, 4, 3, 5, 4, 6, 5, 7, 6, 8, 7, 9, 8, 10, 9, 11, 10, 12, 11, 13],
    numDnsQueries: 6420,
    numBlockedFiltering: 782,
    numReplacedSafebrowsing: 156,
    numReplacedParental: 168,
    numReplacedSafesearch: 45,
    avgProcessingTime: 42,
    topQueriedDomains: [
        { name: 'google.com', count: 1250 },
        { name: 'facebook.com', count: 890 },
        { name: 'youtube.com', count: 756 },
        { name: 'twitter.com', count: 542 },
        { name: 'instagram.com', count: 423 },
        { name: 'reddit.com', count: 312 },
        { name: 'github.com', count: 287 },
        { name: 'stackoverflow.com', count: 198 },
        { name: 'amazon.com', count: 156 },
        { name: 'netflix.com', count: 134 },
    ],
    topBlockedDomains: [
        { name: 'doubleclick.net', count: 245 },
        { name: 'googlesyndication.com', count: 189 },
        { name: 'facebook.com', count: 156 },
        { name: 'analytics.google.com', count: 98 },
        { name: 'ads.yahoo.com', count: 67 },
        { name: 'tracking.example.com', count: 45 },
        { name: 'adserver.example.org', count: 34 },
    ],
    topClients: [
        { name: '192.168.1.10', count: 2340, info: { name: 'MacBook Pro' } },
        { name: '192.168.1.15', count: 1890, info: { name: 'iPhone' } },
        { name: '192.168.1.20', count: 1245, info: { name: 'iPad' } },
        { name: '192.168.1.25', count: 678, info: { name: 'Smart TV' } },
        { name: '192.168.1.30', count: 267, info: null },
    ],
    topUpstreamsResponses: [
        { name: 'https://dns.google/dns-query', count: 3200 },
        { name: 'https://cloudflare-dns.com/dns-query', count: 2100 },
        { name: 'https://dns.quad9.net/dns-query', count: 1120 },
    ],
    topUpstreamsAvgTime: [
        { name: 'https://dns.google/dns-query', count: 35 },
        { name: 'https://cloudflare-dns.com/dns-query', count: 28 },
        { name: 'https://dns.quad9.net/dns-query', count: 45 },
    ],
};

const USE_MOCK_DATA = true; // Set to false to use real data

export const Dashboard = () => {
    const dispatch = useDispatch();
    const { dashboard, stats, access } = useSelector((state: RootState) => state);
    const [protectionMenuOpen, setProtectionMenuOpen] = useState(false);
    const [remainingTime, setRemainingTime] = useState<number | null>(null);

    const {
        protectionEnabled,
        processingProtection,
        protectionDisabledDuration,
    } = dashboard || {};

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
    } = stats || {};

    useEffect(() => {
        dispatch(getStats());
        dispatch(getStatsConfig());
        dispatch(getClients());
        dispatch(getAccessList());
    }, [dispatch]);

    const handleRefreshStats = () => {
        dispatch(getStats());
        dispatch(getStatsConfig());
        dispatch(getClients());
        dispatch(getAccessList());
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
                            numBlockedFiltering={USE_MOCK_DATA ? MOCK_DATA.numBlockedFiltering : (numBlockedFiltering || 0)}
                            numReplacedSafebrowsing={USE_MOCK_DATA ? MOCK_DATA.numReplacedSafebrowsing : (numReplacedSafebrowsing || 0)}
                            numReplacedParental={USE_MOCK_DATA ? MOCK_DATA.numReplacedParental : (numReplacedParental || 0)}
                            dnsQueries={USE_MOCK_DATA ? MOCK_DATA.dnsQueries : (dnsQueries || [])}
                            blockedFiltering={USE_MOCK_DATA ? MOCK_DATA.blockedFiltering : (blockedFiltering || [])}
                            replacedSafebrowsing={USE_MOCK_DATA ? MOCK_DATA.replacedSafebrowsing : (replacedSafebrowsing || [])}
                            replacedParental={USE_MOCK_DATA ? MOCK_DATA.replacedParental : (replacedParental || [])}
                        />

                        <div className={s.statContainer}>
                            <GeneralStatistics
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                                numBlockedFiltering={USE_MOCK_DATA ? MOCK_DATA.numBlockedFiltering : (numBlockedFiltering || 0)}
                                numReplacedSafebrowsing={USE_MOCK_DATA ? MOCK_DATA.numReplacedSafebrowsing : (numReplacedSafebrowsing || 0)}
                                numReplacedParental={USE_MOCK_DATA ? MOCK_DATA.numReplacedParental : (numReplacedParental || 0)}
                                numReplacedSafesearch={USE_MOCK_DATA ? MOCK_DATA.numReplacedSafesearch : (numReplacedSafesearch || 0)}
                                avgProcessingTime={USE_MOCK_DATA ? MOCK_DATA.avgProcessingTime : (avgProcessingTime || 0)}
                                interval={interval || DAY}
                                timeUnits={timeUnits || 'hours'}
                            />

                            <TopClients
                                topClients={USE_MOCK_DATA ? MOCK_DATA.topClients : (topClients || [])}
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                            />

                            <TopQueriedDomains
                                topQueriedDomains={USE_MOCK_DATA ? MOCK_DATA.topQueriedDomains : (topQueriedDomains || [])}
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                            />

                            <TopBlockedDomains
                                topBlockedDomains={USE_MOCK_DATA ? MOCK_DATA.topBlockedDomains : (topBlockedDomains || [])}
                                numBlockedFiltering={USE_MOCK_DATA ? MOCK_DATA.numBlockedFiltering : (numBlockedFiltering || 0)}
                            />

                            <TopUpstreams
                                topUpstreamsResponses={USE_MOCK_DATA ? MOCK_DATA.topUpstreamsResponses : (topUpstreamsResponses || [])}
                                numDnsQueries={USE_MOCK_DATA ? MOCK_DATA.numDnsQueries : (numDnsQueries || 0)}
                            />

                            <UpstreamAvgTime
                                topUpstreamsAvgTime={USE_MOCK_DATA ? MOCK_DATA.topUpstreamsAvgTime : (topUpstreamsAvgTime || [])}
                                avgProcessingTime={USE_MOCK_DATA ? MOCK_DATA.avgProcessingTime : (avgProcessingTime || 0)}
                            />
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};
