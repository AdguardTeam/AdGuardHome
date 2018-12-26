import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import Line from '../ui/Line';

import { getPercent } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';

class Statistics extends Component {
    render() {
        const {
            dnsQueries,
            blockedFiltering,
            replacedSafebrowsing,
            replacedParental,
        } = this.props;

        const filteringData = [this.props.history[1]];
        const queriesData = [this.props.history[2]];
        const parentalData = [this.props.history[3]];
        const safebrowsingData = [this.props.history[4]];

        return (
            <div className="row">
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-blue">
                                {dnsQueries}
                            </div>
                            <div className="card-title-stats">
                                <Trans>dns_query</Trans>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line data={queriesData} color={STATUS_COLORS.blue}/>
                        </div>
                    </Card>
                </div>
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-red">
                                {blockedFiltering}
                            </div>
                            <div className="card-value card-value-percent text-red">
                                {getPercent(dnsQueries, blockedFiltering)}
                            </div>
                            <div className="card-title-stats">
                                <a href="#filters">
                                    <Trans>blocked_by</Trans>
                                </a>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line data={filteringData} color={STATUS_COLORS.red}/>
                        </div>
                    </Card>
                </div>
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-green">
                                {replacedSafebrowsing}
                            </div>
                            <div className="card-value card-value-percent text-green">
                                {getPercent(dnsQueries, replacedSafebrowsing)}
                            </div>
                            <div className="card-title-stats">
                                <Trans>stats_malware_phishing</Trans>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line data={safebrowsingData} color={STATUS_COLORS.green}/>
                        </div>
                    </Card>
                </div>
                <div className="col-sm-6 col-lg-3">
                    <Card type="card--full" bodyType="card-wrap">
                        <div className="card-body-stats">
                            <div className="card-value card-value-stats text-yellow">
                                {replacedParental}
                            </div>
                            <div className="card-value card-value-percent text-yellow">
                                {getPercent(dnsQueries, replacedParental)}
                            </div>
                            <div className="card-title-stats">
                                <Trans>stats_adult</Trans>
                            </div>
                        </div>
                        <div className="card-chart-bg">
                            <Line data={parentalData} color={STATUS_COLORS.yellow}/>
                        </div>
                    </Card>
                </div>
            </div>
        );
    }
}

Statistics.propTypes = {
    history: PropTypes.array.isRequired,
    dnsQueries: PropTypes.number.isRequired,
    blockedFiltering: PropTypes.number.isRequired,
    replacedSafebrowsing: PropTypes.number.isRequired,
    replacedParental: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
};

export default withNamespaces()(Statistics);
