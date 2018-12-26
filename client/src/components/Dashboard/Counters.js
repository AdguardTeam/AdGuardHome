import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import Tooltip from '../ui/Tooltip';

const tooltipType = 'tooltip-custom--narrow';

const Counters = props => (
    <Card title={ props.t('general_statistics') } subtitle={ props.t('for_last_24_hours') } bodyType="card-table" refresh={props.refreshButton}>
        <table className="table card-table">
            <tbody>
                <tr>
                    <td>
                        <Trans>dns_query</Trans>
                        <Tooltip text={ props.t('number_of_dns_query_24_hours') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.dnsQueries}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <a href="#filters">
                            <Trans>blocked_by</Trans>
                        </a>
                        <Tooltip text={ props.t('number_of_dns_query_blocked_24_hours') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.blockedFiltering}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>stats_malware_phishing</Trans>
                        <Tooltip text={ props.t('number_of_dns_query_blocked_24_hours_by_sec') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedSafebrowsing}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>stats_adult</Trans>
                        <Tooltip text={ props.t('number_of_dns_query_blocked_24_hours_adult') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedParental}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>enforced_save_search</Trans>
                        <Tooltip text={ props.t('number_of_dns_query_to_safe_search') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedSafesearch}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>average_processing_time</Trans>
                        <Tooltip text={ props.t('average_processing_time_hint') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.avgProcessingTime}
                        </span>
                    </td>
                </tr>
            </tbody>
        </table>
    </Card>
);

Counters.propTypes = {
    dnsQueries: PropTypes.number.isRequired,
    blockedFiltering: PropTypes.number.isRequired,
    replacedSafebrowsing: PropTypes.number.isRequired,
    replacedParental: PropTypes.number.isRequired,
    replacedSafesearch: PropTypes.number.isRequired,
    avgProcessingTime: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Counters);
