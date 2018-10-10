import React from 'react';
import PropTypes from 'prop-types';

import Card from '../ui/Card';
import Tooltip from '../ui/Tooltip';

const tooltipType = 'tooltip-custom--narrow';

const Counters = props => (
    <Card title="General statistics" subtitle="for the last 24 hours" bodyType="card-table" refresh={props.refreshButton}>
        <table className="table card-table">
            <tbody>
                <tr>
                    <td>
                        DNS Queries
                        <Tooltip text="A number of DNS quieries processed for the last 24 hours" type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.dnsQueries}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        Blocked by <a href="#filters">Filters</a>
                        <Tooltip text="A number of DNS requests blocked by adblock filters and hosts blocklists" type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.blockedFiltering}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        Blocked malware/phishing
                        <Tooltip text="A number of DNS requests blocked by the AdGuard browsing security module" type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedSafebrowsing}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        Blocked adult websites
                        <Tooltip text="A number of adult websites blocked" type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedParental}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        Enforced safe search
                        <Tooltip text="A number of DNS requests to search engines for which Safe Search was enforced" type={tooltipType} />
                    </td>
                    <td className="text-right">
                        <span className="text-muted">
                            {props.replacedSafesearch}
                        </span>
                    </td>
                </tr>
                <tr>
                    <td>
                        Average processing time
                        <Tooltip text="Average time in milliseconds on processing a DNS request" type={tooltipType} />
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
    refreshButton: PropTypes.node,
};

export default Counters;
