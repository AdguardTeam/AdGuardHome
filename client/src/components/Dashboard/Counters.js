import React from 'react';
import propTypes from 'prop-types';
import { Trans, useTranslation } from 'react-i18next';
import round from 'lodash/round';
import { shallowEqual, useSelector } from 'react-redux';
import Card from '../ui/Card';
import { formatNumber, msToDays, msToHours } from '../../helpers/helpers';
import LogsSearchLink from '../ui/LogsSearchLink';
import { RESPONSE_FILTER, TIME_UNITS } from '../../helpers/constants';
import Tooltip from '../ui/Tooltip';

const Row = ({
    label, count, response_status, tooltipTitle, translationComponents,
}) => {
    const content = response_status
        ? <LogsSearchLink response_status={response_status}>{formatNumber(count)}</LogsSearchLink>
        : count;

    return (
        <div className="counters__row" key={label}>
            <div className="counters__column">
                <span className="counters__title">
                    <Trans components={translationComponents}>
                        {label}
                    </Trans>
                </span>
                <span className="counters__tooltip">
                    <Tooltip
                        content={tooltipTitle}
                        placement="top"
                        className="tooltip-container tooltip-custom--narrow text-center"
                    >
                        <svg className="icons icon--20 icon--lightgray ml-2">
                            <use xlinkHref="#question" />
                        </svg>
                    </Tooltip>
                </span>
            </div>
            <div className="counters__column counters__column--value">
                <strong>{content}</strong>
            </div>
        </div>
    );
};

const Counters = ({ refreshButton, subtitle }) => {
    const {
        interval,
        numDnsQueries,
        numBlockedFiltering,
        numReplacedSafebrowsing,
        numReplacedParental,
        numReplacedSafesearch,
        avgProcessingTime,
        timeUnits,
    } = useSelector((state) => state.stats, shallowEqual);
    const { t } = useTranslation();

    const dnsQueryTooltip = timeUnits === TIME_UNITS.HOURS
        ? t('number_of_dns_query_hours', { count: msToHours(interval) })
        : t('number_of_dns_query_days', { count: msToDays(interval) });

    const rows = [
        {
            label: 'dns_query',
            count: numDnsQueries,
            tooltipTitle: dnsQueryTooltip,
            response_status: RESPONSE_FILTER.ALL.QUERY,
        },
        {
            label: 'blocked_by',
            count: numBlockedFiltering,
            tooltipTitle: 'number_of_dns_query_blocked_24_hours',
            response_status: RESPONSE_FILTER.BLOCKED.QUERY,
            translationComponents: [<a href="#filters" key="0">link</a>],
        },
        {
            label: 'stats_malware_phishing',
            count: numReplacedSafebrowsing,
            tooltipTitle: 'number_of_dns_query_blocked_24_hours_by_sec',
            response_status: RESPONSE_FILTER.BLOCKED_THREATS.QUERY,
        },
        {
            label: 'stats_adult',
            count: numReplacedParental,
            tooltipTitle: 'number_of_dns_query_blocked_24_hours_adult',
            response_status: RESPONSE_FILTER.BLOCKED_ADULT_WEBSITES.QUERY,
        },
        {
            label: 'enforced_save_search',
            count: numReplacedSafesearch,
            tooltipTitle: 'number_of_dns_query_to_safe_search',
            response_status: RESPONSE_FILTER.SAFE_SEARCH.QUERY,
        },
        {
            label: 'average_processing_time',
            count: avgProcessingTime ? `${round(avgProcessingTime)} ms` : 0,
            tooltipTitle: 'average_processing_time_hint',
        },
    ];

    return (
        <Card
            title={t('general_statistics')}
            subtitle={subtitle}
            bodyType="card-table"
            refresh={refreshButton}
        >
            <div className="counters">
                {rows.map(Row)}
            </div>
        </Card>
    );
};

Row.propTypes = {
    label: propTypes.string.isRequired,
    count: propTypes.string.isRequired,
    response_status: propTypes.string,
    tooltipTitle: propTypes.string.isRequired,
    translationComponents: propTypes.arrayOf(propTypes.element),
};

Counters.propTypes = {
    refreshButton: propTypes.node.isRequired,
    subtitle: propTypes.string.isRequired,
};

export default Counters;
