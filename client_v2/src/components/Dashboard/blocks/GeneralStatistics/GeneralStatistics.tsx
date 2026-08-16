import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { EmptyState } from '../EmptyState';

import s from './GeneralStatistics.module.pcss';
import { StatRow } from '../StatRow';
import { formatCompactNumber } from 'panel/helpers/helpers';

type Props = {
    numDnsQueries: number;
    numBlockedFiltering: number;
    numReplacedSafebrowsing: number;
    numReplacedParental: number;
    numReplacedSafesearch: number;
    avgProcessingTime: number;
};

export const GeneralStatistics = (props: Props) => {
    const blockedPercent = () =>
        props.numDnsQueries > 0 ? (props.numBlockedFiltering / props.numDnsQueries) * 100 : 0;
    const safebrowsingPercent = () =>
        props.numDnsQueries > 0 ? (props.numReplacedSafebrowsing / props.numDnsQueries) * 100 : 0;
    const parentalPercent = () =>
        props.numDnsQueries > 0 ? (props.numReplacedParental / props.numDnsQueries) * 100 : 0;
    const safesearchPercent = () =>
        props.numDnsQueries > 0 ? (props.numReplacedSafesearch / props.numDnsQueries) * 100 : 0;

    const hasStats = () => props.numDnsQueries > 0;

    return (
        <div class={s.card}>
            <div class={s.cardHeader}>
                <div class={cn(theme.title.h5, s.cardTitle)}>
                    {intl.getMessage('general_statistics')}
                </div>

                <Show when={hasStats()}>
                    <div class={cn(theme.text.t3, s.cardSubtitle)}>
                        {intl.getPlural('queries_total', props.numDnsQueries, {
                            value: formatCompactNumber(props.numDnsQueries),
                        })}
                    </div>
                </Show>
            </div>

            <Show when={hasStats()} fallback={<EmptyState />}>
                <div class={s.tableRows}>
                    <StatRow
                        label={intl.getMessage('dns_queries')}
                        value={props.numDnsQueries}
                        icon="connections"
                        rowTheme="dnsQueries"
                        tooltip={intl.getMessage('dns_queries_tooltip')}
                        isTotal
                    />

                    <StatRow
                        label={intl.getMessage('ads_blocked')}
                        value={props.numBlockedFiltering}
                        percent={blockedPercent()}
                        icon="adblocking"
                        rowTheme="adsBlocked"
                        tooltip={intl.getMessage('ads_blocked_tooltip')}
                    />

                    <StatRow
                        label={intl.getMessage('threats_blocked')}
                        value={props.numReplacedSafebrowsing}
                        percent={safebrowsingPercent()}
                        icon="tracking"
                        rowTheme="threatsBlocked"
                        tooltip={intl.getMessage('threats_blocked_tooltip')}
                    />

                    <StatRow
                        label={intl.getMessage('adult_websites_blocked')}
                        value={props.numReplacedParental}
                        percent={parentalPercent()}
                        icon="parental"
                        rowTheme="adultWebsitesBlocked"
                        tooltip={intl.getMessage('adult_websites_blocked_tooltip')}
                    />

                    <StatRow
                        label={intl.getMessage('safe_search_used')}
                        value={props.numReplacedSafesearch}
                        percent={safesearchPercent()}
                        icon="search"
                        rowTheme="safeSearchUsed"
                        tooltip={intl.getMessage('safe_search_used_tooltip')}
                    />

                    <div class={s.rowDivider} />

                    <div class={s.processingTimeRow}>
                        <StatRow
                            label={intl.getMessage('average_time_processing')}
                            value={intl.getMessage('processing_time_ms', {
                                value: (props.avgProcessingTime ?? 0).toFixed(0),
                            })}
                            isQueriesValue={false}
                            icon="time"
                            rowTheme="averageProcessingTime"
                            tooltip={intl.getMessage('average_time_processing_tooltip')}
                        />
                    </div>
                </div>
            </Show>
        </div>
    );
};
