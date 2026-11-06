import { createMemo } from 'solid-js';

import intl from 'panel/common/intl';
import { RoutePath } from 'panel/components/Routes/Paths';
import { resolveCssColor } from 'panel/helpers/colors';
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

const CARD_COLOR_TOKENS = {
    [CARDS_THEME.QUERIES]: { token: '--default-product-icon', fallback: '#67b279' },
    [CARDS_THEME.ADS]: { token: '--default-error-link', fallback: '#e9653a' },
    [CARDS_THEME.THREATS]: { token: '--default-attention-link', fallback: '#d58500' },
    [CARDS_THEME.ADULT]: { token: '--default-additional-link', fallback: '#a870b2' },
};

export const StatCards = (props: Props) => {
    const isDark = useIsDarkTheme();

    const colors = createMemo(() => {
        // Read the theme signal so the computed values refresh on a theme switch.
        isDark();

        return {
            [CARDS_THEME.QUERIES]: resolveCssColor(
                CARD_COLOR_TOKENS[CARDS_THEME.QUERIES].token,
                CARD_COLOR_TOKENS[CARDS_THEME.QUERIES].fallback,
            ),
            [CARDS_THEME.ADS]: resolveCssColor(
                CARD_COLOR_TOKENS[CARDS_THEME.ADS].token,
                CARD_COLOR_TOKENS[CARDS_THEME.ADS].fallback,
            ),
            [CARDS_THEME.THREATS]: resolveCssColor(
                CARD_COLOR_TOKENS[CARDS_THEME.THREATS].token,
                CARD_COLOR_TOKENS[CARDS_THEME.THREATS].fallback,
            ),
            [CARDS_THEME.ADULT]: resolveCssColor(
                CARD_COLOR_TOKENS[CARDS_THEME.ADULT].token,
                CARD_COLOR_TOKENS[CARDS_THEME.ADULT].fallback,
            ),
        };
    });

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
