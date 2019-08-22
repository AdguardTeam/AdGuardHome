import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import round from 'lodash/round';

import Card from '../ui/Card';
import Tooltip from '../ui/Tooltip';

const tooltipType = 'tooltip-custom--narrow';

const Counters = (props) => {
    const {
        t,
        interval,
        refreshButton,
        subtitle,
        dnsQueries,
        blockedFiltering,
        replacedSafebrowsing,
        replacedParental,
        replacedSafesearch,
        avgProcessingTime,
    } = props;

    const tooltipTitle =
        interval === 1
            ? t('number_of_dns_query_24_hours')
            : t('number_of_dns_query_days', { value: interval });

    return (
        <Card
            title={t('general_statistics')}
            subtitle={subtitle}
            bodyType="card-table"
            refresh={refreshButton}
        >
            <table className="table card-table">
                <tbody>
                    <tr>
                        <td>
                            <Trans>dns_query</Trans>
                            <Tooltip text={tooltipTitle} type={tooltipType} />
                        </td>
                        <td className="text-right">
                            <span className="text-muted">{dnsQueries}</span>
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <a href="#filters">
                                <Trans>blocked_by</Trans>
                            </a>
                            <Tooltip
                                text={t('number_of_dns_query_blocked_24_hours')}
                                type={tooltipType}
                            />
                        </td>
                        <td className="text-right">
                            <span className="text-muted">{blockedFiltering}</span>
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <Trans>stats_malware_phishing</Trans>
                            <Tooltip
                                text={t('number_of_dns_query_blocked_24_hours_by_sec')}
                                type={tooltipType}
                            />
                        </td>
                        <td className="text-right">
                            <span className="text-muted">{replacedSafebrowsing}</span>
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <Trans>stats_adult</Trans>
                            <Tooltip
                                text={t('number_of_dns_query_blocked_24_hours_adult')}
                                type={tooltipType}
                            />
                        </td>
                        <td className="text-right">
                            <span className="text-muted">{replacedParental}</span>
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <Trans>enforced_save_search</Trans>
                            <Tooltip
                                text={t('number_of_dns_query_to_safe_search')}
                                type={tooltipType}
                            />
                        </td>
                        <td className="text-right">
                            <span className="text-muted">{replacedSafesearch}</span>
                        </td>
                    </tr>
                    <tr>
                        <td>
                            <Trans>average_processing_time</Trans>
                            <Tooltip text={t('average_processing_time_hint')} type={tooltipType} />
                        </td>
                        <td className="text-right">
                            <span className="text-muted">
                                {avgProcessingTime ? `${round(avgProcessingTime, 2)} ms` : 0}
                            </span>
                        </td>
                    </tr>
                </tbody>
            </table>
        </Card>
    );
};

Counters.propTypes = {
    dnsQueries: PropTypes.number.isRequired,
    blockedFiltering: PropTypes.number.isRequired,
    replacedSafebrowsing: PropTypes.number.isRequired,
    replacedParental: PropTypes.number.isRequired,
    replacedSafesearch: PropTypes.number.isRequired,
    avgProcessingTime: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
    interval: PropTypes.number.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Counters);
