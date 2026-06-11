import React, { useEffect, useState, useMemo, useRef, useCallback } from 'react';
import { useSelector, useDispatch } from 'react-redux';

import theme from 'panel/lib/theme';
import { RootState } from 'panel/initialState';
import { Loader, PageLoader } from 'panel/common/ui/Loader';
import { toggleProtection, getClients } from 'panel/actions';
import { getStats, getStatsConfig } from 'panel/actions/stats';
import { getAccessList } from 'panel/actions/access';
import { ONE_SECOND_IN_MS, HOUR, DAY, STATS_INTERVALS_DAYS } from 'panel/helpers/constants';

import { Header, getPeriodLabel } from './blocks/Header/Header';
import { StatCards } from './blocks/StatCards';
import { GeneralStatistics } from './blocks/GeneralStatistics';
import { TopClients } from './blocks/TopClients';
import { TopQueriedDomains } from './blocks/TopQueriedDomains';
import { TopBlockedDomains } from './blocks/TopBlockedDomains';
import { TopUpstreams } from './blocks/TopUpstreams';
import { UpstreamAvgTime } from './blocks/UpstreamAvgTime';

import s from './Dashboard.module.pcss';

export const Dashboard = () => {
    const dispatch = useDispatch();
    const { dashboard, stats, access } = useSelector((state: RootState) => state);
    const [remainingTime, setRemainingTime] = useState<number | null>(null);
    const [selectedPeriod, setSelectedPeriod] = useState(DAY);
    const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

    const protectionDisabledDuration = dashboard?.protectionDisabledDuration;

    const startCountdown = useCallback(
        (duration: number) => {
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
        },
        [dispatch],
    );

    useEffect(() => {
        if (
            protectionDisabledDuration &&
            protectionDisabledDuration > 0 &&
            timerRef.current === null
        ) {
            startCountdown(protectionDisabledDuration);
        }
    }, [protectionDisabledDuration, startCountdown]);

    useEffect(
        () => () => {
            if (timerRef.current) {
                clearInterval(timerRef.current);
            }
        },
        [],
    );

    const maxStatsInterval = stats?.interval || DAY;
    const effectiveMaxStatsInterval = maxStatsInterval >= HOUR ? maxStatsInterval : DAY;
    const periodIntervals = useMemo(() => {
        const intervals = STATS_INTERVALS_DAYS.filter(
            (interval) => interval <= effectiveMaxStatsInterval,
        );

        if (!intervals.includes(effectiveMaxStatsInterval)) {
            intervals.push(effectiveMaxStatsInterval);
        }

        return intervals.sort((a, b) => a - b);
    }, [effectiveMaxStatsInterval]);

    const periodOptions = useMemo(
        () =>
            periodIntervals.map((interval) => ({
                value: interval,
                label: getPeriodLabel(interval),
            })),
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

    const { protectionEnabled, processingProtection } = dashboard;

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

    const handleToggleProtection = useCallback(
        (enabled: boolean, duration?: number) => {
            if (!enabled && timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
                setRemainingTime(null);
            }
            dispatch(toggleProtection(enabled, duration));
        },
        [dispatch],
    );

    const handlePeriodChange = (period: number) => {
        setSelectedPeriod(period);
    };

    const isLoading = processingStats || processingGetConfig || access?.processing;

    return (
        <div className={theme.layout.container}>
            <div className={theme.layout.containerIn}>
                <Header
                    protectionEnabled={!!protectionEnabled}
                    processingProtection={processingProtection}
                    remainingTime={remainingTime}
                    selectedPeriod={selectedPeriod}
                    periodOptions={periodOptions}
                    isLoading={isLoading}
                    onToggleProtection={handleToggleProtection}
                    onRefreshStats={handleRefreshStats}
                    onPeriodChange={handlePeriodChange}
                />

                {isLoading ? (
                    <div className={s.loader}>
                        <PageLoader />
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

                            <TopClients topClients={topClients} numDnsQueries={numDnsQueries} />

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
