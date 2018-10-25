import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import Tooltip from '../ui/Tooltip';

const tooltipType = 'tooltip-custom--narrow';

const Counters = props => (
    <Card title={ props.t('General statistics') } subtitle={ props.t('for the last 24 hours') } bodyType="card-table" refresh={props.refreshButton}>
        <table className="table card-table">
            <tbody>
                <tr>
                    <td>
                        <Trans>DNS Queries</Trans>
                        <Tooltip text={ props.t('A number of DNS quieries processed for the last 24 hours') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.dnsQueries}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>Blocked by</Trans> <a href="#filters"><Trans>Filters</Trans></a>
                        <Tooltip text={ props.t('A number of DNS requests blocked by adblock filters and hosts blocklists') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.blockedFiltering}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>Blocked malware/phishing</Trans>
                        <Tooltip text={ props.t('A number of DNS requests blocked by the AdGuard browsing security module') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedSafebrowsing}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>Blocked adult websites</Trans>
                        <Tooltip text={ props.t('A number of adult websites blocked') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedParental}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>Enforced safe search</Trans>
                        <Tooltip text={ props.t('A number of DNS requests to search engines for which Safe Search was enforced') } type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedSafesearch}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        <Trans>Average processing time</Trans>
                        <Tooltip text={ props.t('Average time in milliseconds on processing a DNS request') } type={tooltipType} />
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
    t: PropTypes.func,
};

export default withNamespaces()(Counters);
