import {
    createSignal,
    createMemo,
    createEffect,
    onCleanup,
    onMount,
    Show,
} from 'solid-js';

import theme from 'panel/lib/theme';
import { PageLoader } from 'panel/common/ui/Loader';
import { dashboardState, toggleProtection, getClients } from 'panel/stores/dashboard';
import { statsState, getStats, getStatsConfig, enableStatistics } from 'panel/stores/stats';
import { accessState, getAccessList } from 'panel/stores/access';
import { getStoredStatsPeriod, getClampedMaxInterval } from 'panel/helpers/statistics';
import { LocalStorageHelper, LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { ONE_SECOND_IN_MS, DAY, STATS_INTERVALS_DAYS } from 'panel/helpers/constants';

import { Header, getPeriodLabel } from './blocks/Header/Header';
import { StatCards } from './blocks/StatCards';
import { EmptyState } from './blocks/EmptyState/EmptyState';
import { GeneralStatistics } from './blocks/GeneralStatistics';
import { TopClients } from './blocks/TopClients';
import { TopQueriedDomains } from './blocks/TopQueriedDomains';
import { TopBlockedDomains } from './blocks/TopBlockedDomains';
import { TopUpstreams } from './blocks/TopUpstreams';
import { UpstreamAvgTime } from './blocks/UpstreamAvgTime';

import s from './Dashboard.module.pcss';

export const Dashboard = () => {
    const [remainingTime, setRemainingTime] = createSignal<number | null>(null);
    const [selectedPeriod, setSelectedPeriod] = createSignal(getStoredStatsPeriod());
    let timerRef: ReturnType<typeof setInterval> | null = null;

    const startCountdown = (duration: number) => {
        if (timerRef) {
            clearInterval(timerRef);
        }
        setRemainingTime(duration);
        timerRef = setInterval(() => {
            const prev = remainingTime();
            if (prev !== null && prev > ONE_SECOND_IN_MS) {
                setRemainingTime(prev - ONE_SECOND_IN_MS);
            } else {
                if (timerRef) {
                    clearInterval(timerRef);
                    timerRef = null;
                }
                toggleProtection(null);
                setRemainingTime(null);
            }
        }, ONE_SECOND_IN_MS);
    };

    createEffect(() => {
        const protectionDisabledDuration = dashboardState.protectionDisabledDuration;
        if (protectionDisabledDuration && protectionDisabledDuration > 0 && timerRef === null) {
            startCountdown(protectionDisabledDuration);
        }
    });

    onCleanup(() => {
        if (timerRef) {
            clearInterval(timerRef);
        }
    });

    const maxStatsInterval = createMemo(() => getClampedMaxInterval(statsState.interval));

    const periodOptions = createMemo(() => {
        const max = maxStatsInterval();
        const intervals = STATS_INTERVALS_DAYS.filter((interval) => interval <= max);

        if (!intervals.includes(max)) {
            intervals.push(max);
        }

        // Until the first config attempt settles the list is built from the
        // default DAY, so keep the stored period selectable to avoid showing a
        // period the user did not pick.
        const storedPeriod = selectedPeriod();
        if (!statsState.configAttempted && !intervals.includes(storedPeriod)) {
            intervals.push(storedPeriod);
        }

        return intervals
            .sort((a, b) => a - b)
            .map((interval) => ({ value: interval, label: getPeriodLabel(interval) }));
    });

    const effectivePeriod = createMemo(() => {
        const options = periodOptions();
        return Math.min(selectedPeriod(), options[options.length - 1]?.value ?? DAY);
    });

    onMount(() => {
        getClients();
        getAccessList();
        getStatsConfig();
    });

    createEffect(() => {
        const period = selectedPeriod();
        // No stats request before the first config attempt settles: it would
        // be clamped by the default DAY and paint 24h values before the real
        // ones.  Tracks `selectedPeriod` only; the interval/configLoaded
        // changes alone must not re-fire the request.
        if (!statsState.configAttempted) {
            return;
        }
        // `getStats` clamps the period by the real server retention itself.
        getStats(period);
    });

    const handleRefreshStats = () => {
        getClients();
        getAccessList();
        getStatsConfig();
        getStats(selectedPeriod());
    };

    const handleToggleProtection = (enabled: boolean, duration?: number) => {
        if (!enabled && timerRef) {
            clearInterval(timerRef);
            timerRef = null;
            setRemainingTime(null);
        }
        toggleProtection(enabled ? duration : null);
    };

    const handlePeriodChange = (period: number) => {
        setSelectedPeriod(period);
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.STATS_PERIOD, period);
    };

    const isLoading = () =>
        statsState.processingStats || statsState.processingGetConfig || accessState.processing;

    // The page loader shows exactly once: until the first stats request
    // settles.  Later refetches keep the previous values painted, so the
    // loader can never flash back in.
    const isInitialLoading = () => !statsState.statsAttempted;

    return (
        <div class={theme.layout.container}>
            <div class={theme.layout.containerIn}>
                <Header
                    protectionEnabled={!!dashboardState.protectionEnabled}
                    processingProtection={dashboardState.processingProtection}
                    remainingTime={remainingTime()}
                    selectedPeriod={effectivePeriod()}
                    periodOptions={periodOptions()}
                    isLoading={isLoading()}
                    onToggleProtection={handleToggleProtection}
                    onRefreshStats={handleRefreshStats}
                    onPeriodChange={handlePeriodChange}
                />

                <Show
                    when={!isInitialLoading()}
                    fallback={
                        <div class={s.loader}>
                            <PageLoader />
                        </div>
                    }
                >
                    <StatCards
                        numDnsQueries={statsState.numDnsQueries}
                        numBlockedFiltering={statsState.numBlockedFiltering}
                        numReplacedSafebrowsing={statsState.numReplacedSafebrowsing}
                        numReplacedParental={statsState.numReplacedParental}
                        dnsQueries={statsState.dnsQueries}
                        blockedFiltering={statsState.blockedFiltering}
                        replacedSafebrowsing={statsState.replacedSafebrowsing}
                        replacedParental={statsState.replacedParental}
                        timeUnits={statsState.timeUnits}
                    />

                    <Show
                        when={statsState.enabled}
                        fallback={
                            <EmptyState
                                mode="disabled"
                                class={s.emptyState}
                                onEnable={() => enableStatistics(effectivePeriod())}
                            />
                        }
                    >
                        <div class={s.statContainer}>
                            <GeneralStatistics
                                numDnsQueries={statsState.numDnsQueries}
                                numBlockedFiltering={statsState.numBlockedFiltering}
                                numReplacedSafebrowsing={statsState.numReplacedSafebrowsing}
                                numReplacedParental={statsState.numReplacedParental}
                                numReplacedSafesearch={statsState.numReplacedSafesearch}
                                avgProcessingTime={statsState.avgProcessingTime}
                            />

                            <TopClients
                                topClients={statsState.topClients}
                                numDnsQueries={statsState.numDnsQueries}
                                period={effectivePeriod()}
                            />

                            <TopQueriedDomains
                                topQueriedDomains={statsState.topQueriedDomains}
                                numDnsQueries={statsState.numDnsQueries}
                                period={effectivePeriod()}
                            />

                            <TopBlockedDomains
                                topBlockedDomains={statsState.topBlockedDomains}
                                numBlockedFiltering={statsState.numBlockedFiltering}
                                period={effectivePeriod()}
                            />

                            <TopUpstreams
                                topUpstreamsResponses={statsState.topUpstreamsResponses}
                                numDnsQueries={statsState.numDnsQueries}
                                period={effectivePeriod()}
                            />

                            <UpstreamAvgTime
                                topUpstreamsAvgTime={statsState.topUpstreamsAvgTime}
                                avgProcessingTime={statsState.avgProcessingTime}
                                period={effectivePeriod()}
                            />
                        </div>
                    </Show>
                </Show>
            </div>
        </div>
    );
};
