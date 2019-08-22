import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import Line from '../ui/Line';

import { getPercent, normalizeHistory } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';

class Statistics extends Component {
    getNormalizedHistory = (data, interval, id) => [{ data: normalizeHistory(data, interval), id }];

    render() {
        const {
            interval,
            dnsQueries,
            blockedFiltering,
            replacedSafebrowsing,
            replacedParental,
            numDnsQueries,
            numBlockedFiltering,
            numReplacedSafebrowsing,
            numReplacedParental,
        } = this.props;

        return (
            <div className="row">
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-blue">
                                {numDnsQueries}
                            </div>
                            <div className="card-title-stats">
                                <Trans>dns_query</Trans>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line
                                data={this.getNormalizedHistory(dnsQueries, interval, 'dnsQueries')}
                                color={STATUS_COLORS.blue}
                            />
                        </div>
                    </Card>
                </div>
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-red">
                                {numBlockedFiltering}
                            </div>
                            <div className="card-value card-value-percent text-red">
                                {getPercent(numDnsQueries, numBlockedFiltering)}
                            </div>
                            <div className="card-title-stats">
                                <a href="#filters">
                                    <Trans>blocked_by</Trans>
                                </a>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line
                                data={this.getNormalizedHistory(
                                    blockedFiltering,
                                    interval,
                                    'blockedFiltering',
                                )}
                                color={STATUS_COLORS.red}
                            />
                        </div>
                    </Card>
                </div>
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-green">
                                {numReplacedSafebrowsing}
                            </div>
                            <div className="card-value card-value-percent text-green">
                                {getPercent(numDnsQueries, numReplacedSafebrowsing)}
                            </div>
                            <div className="card-title-stats">
                                <Trans>stats_malware_phishing</Trans>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line
                                data={this.getNormalizedHistory(
                                    replacedSafebrowsing,
                                    interval,
                                    'replacedSafebrowsing',
                                )}
                                color={STATUS_COLORS.green}
                            />
                        </div>
                    </Card>
                </div>
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-yellow">
                                {numReplacedParental}
                            </div>
                            <div className="card-value card-value-percent text-yellow">
                                {getPercent(numDnsQueries, numReplacedParental)}
                            </div>
                            <div className="card-title-stats">
                                <Trans>stats_adult</Trans>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line
                                data={this.getNormalizedHistory(
                                    replacedParental,
                                    interval,
                                    'replacedParental',
                                )}
                                color={STATUS_COLORS.yellow}
                            />
                        </div>
                    </Card>
                </div>
            </div>
        );
    }
}

Statistics.propTypes = {
    interval: PropTypes.number.isRequired,
    dnsQueries: PropTypes.array.isRequired,
    blockedFiltering: PropTypes.array.isRequired,
    replacedSafebrowsing: PropTypes.array.isRequired,
    replacedParental: PropTypes.array.isRequired,
    numDnsQueries: PropTypes.number.isRequired,
    numBlockedFiltering: PropTypes.number.isRequired,
    numReplacedSafebrowsing: PropTypes.number.isRequired,
    numReplacedParental: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
};

export default withNamespaces()(Statistics);
