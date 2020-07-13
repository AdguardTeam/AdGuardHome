import React from 'react';
import propTypes from 'prop-types';
import { Trans, useTranslation } from 'react-i18next';
import round from 'lodash/round';
import { shallowEqual, useSelector } from 'react-redux';
import Card from '../ui/Card';
import IconTooltip from '../ui/IconTooltip';
import { formatNumber } from '../../helpers/helpers';
import LogsSearchLink from '../ui/LogsSearchLink';
import { RESPONSE_FILTER } from '../../helpers/constants';

const Row = ({
    label, count, response_status, tooltipTitle, translationComponents,
}) => {
    const content = response_status
        ? <LogsSearchLink response_status={response_status}>{formatNumber(count)}</LogsSearchLink>
        : count;

    return <tr key={label}>
        <td>
            <Trans components={translationComponents}>{label}</Trans>
            <IconTooltip text={tooltipTitle} type="tooltip-custom--narrow" />
        </td>
        <td className="text-right"><strong>{content}</strong></td>
    </tr>;
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
    } = useSelector((state) => state.stats, shallowEqual);
    const { t } = useTranslation();

    const rows = [
        {
            label: 'dns_query',
            count: numDnsQueries,
            tooltipTitle: interval === 1 ? 'number_of_dns_query_24_hours' : t('number_of_dns_query_days', { count: interval }),
            response_status: RESPONSE_FILTER.ALL.query,
        },
        {
            label: 'blocked_by',
            count: numBlockedFiltering,
            tooltipTitle: 'number_of_dns_query_blocked_24_hours',
            response_status: RESPONSE_FILTER.BLOCKED.query,
            translationComponents: [<a href="#filters" key="0">link</a>],
        },
        {
            label: 'stats_malware_phishing',
            count: numReplacedSafebrowsing,
            tooltipTitle: 'number_of_dns_query_blocked_24_hours_by_sec',
            response_status: RESPONSE_FILTER.BLOCKED_THREATS.query,
        },
        {
            label: 'stats_adult',
            count: numReplacedParental,
            tooltipTitle: 'number_of_dns_query_blocked_24_hours_adult',
            response_status: RESPONSE_FILTER.BLOCKED_ADULT_WEBSITES.query,
        },
        {
            label: 'enforced_save_search',
            count: numReplacedSafesearch,
            tooltipTitle: 'number_of_dns_query_to_safe_search',
            response_status: RESPONSE_FILTER.SAFE_SEARCH.query,
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
            <table className="table card-table">
                <tbody>{rows.map(Row)}</tbody>
            </table>
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
