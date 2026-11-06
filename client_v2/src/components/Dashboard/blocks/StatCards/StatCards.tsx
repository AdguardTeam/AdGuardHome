import { createMemo } from 'solid-js';

import intl from 'panel/common/intl';
import { RoutePath } from 'panel/components/Routes/Paths';
import { QUERY_LOG_REASON_FILTER, QUERY_LOG_STATUS_FILTER } from 'panel/helpers/constants';
import { useIsDarkTheme } from 'panel/hooks/useTheme';

import s from '../StatCard/StatCard.module.pcss';
import { StatCard, CARDS_THEME } from '../StatCard';

type Props = {
    numDnsQueries: number;
    numBlockedFiltering: number;
    numReplacedSafebrowsing: number;
    numReplacedParental: number;
    dnsQueries: number[];
    blockedFiltering: number[];
    replacedSafebrowsing: number[];
    replacedParental: number[];
    timeUnits: string;
};

/**
 * Chart line colors per theme.  Chart.js paints into a canvas, which cannot
 * consume CSS custom properties, so the chart palette is repeated here; keep it
 * in sync with the `--chart-*` tokens in `common/styles/colors/light.css`.
 */
const CHART_COLORS = {
    light: {
        [CARDS_THEME.QUERIES]: '#67b279',
        [CARDS_THEME.ADS]: '#e9653a',
        [CARDS_THEME.THREATS]: '#d58500',
        [CARDS_THEME.ADULT]: '#a870b2',
    },
    dark: {
        [CARDS_THEME.QUERIES]: '#67b279',
        [CARDS_THEME.ADS]: '#f67247',
        [CARDS_THEME.THREATS]: '#d58500',
        [CARDS_THEME.ADULT]: '#a870b2',
    },
};

export const StatCards = (props: Props) => {
    const isDark = useIsDarkTheme();

    const colors = createMemo(() => (isDark() ? CHART_COLORS.dark : CHART_COLORS.light));

    const blockedPercent = () =>
        props.numDnsQueries > 0 ? (props.numBlockedFiltering / props.numDnsQueries) * 100 : 0;
    const threatsPercent = () =>
        props.numDnsQueries > 0 ? (props.numReplacedSafebrowsing / props.numDnsQueries) * 100 : 0;
    const parentalPercent = () =>
        props.numDnsQueries > 0 ? (props.numReplacedParental / props.numDnsQueries) * 100 : 0;

    return (
        <div class={s.statsCards}>
            <StatCard
                value={props.numDnsQueries}
                label={intl.getMessage('dns_query')}
                data={props.dnsQueries}
                timeUnits={props.timeUnits}
                color={colors()[CARDS_THEME.QUERIES]}
                cardTheme={CARDS_THEME.QUERIES}
                linkTo={RoutePath.QueryLog}
            />
            <StatCard
                value={props.numBlockedFiltering}
                label={intl.getMessage('ads_blocked_card')}
                data={props.blockedFiltering}
                timeUnits={props.timeUnits}
                color={colors()[CARDS_THEME.ADS]}
                percentValue={blockedPercent()}
                cardTheme={CARDS_THEME.ADS}
                linkTo={RoutePath.QueryLog}
                query={{
                    status: QUERY_LOG_STATUS_FILTER.BLOCKED.QUERY,
                    reason: QUERY_LOG_REASON_FILTER.BLOCKED_BY_FILTER.QUERY,
                }}
            />
            <StatCard
                value={props.numReplacedSafebrowsing}
                label={intl.getMessage('blocked_threats_chart')}
                data={props.replacedSafebrowsing}
                timeUnits={props.timeUnits}
                color={colors()[CARDS_THEME.THREATS]}
                percentValue={threatsPercent()}
                cardTheme={CARDS_THEME.THREATS}
                linkTo={RoutePath.QueryLog}
                query={{
                    status: QUERY_LOG_STATUS_FILTER.BLOCKED.QUERY,
                    reason: QUERY_LOG_REASON_FILTER.BLOCKED_BY_THREATS.QUERY,
                }}
            />
            <StatCard
                value={props.numReplacedParental}
                label={intl.getMessage('stats_adult')}
                data={props.replacedParental}
                timeUnits={props.timeUnits}
                color={colors()[CARDS_THEME.ADULT]}
                percentValue={parentalPercent()}
                cardTheme={CARDS_THEME.ADULT}
                linkTo={RoutePath.QueryLog}
                query={{
                    status: QUERY_LOG_STATUS_FILTER.BLOCKED.QUERY,
                    reason: QUERY_LOG_REASON_FILTER.BLOCKED_BY_PARENTAL_CONTROL.QUERY,
                }}
            />
        </div>
    );
};
