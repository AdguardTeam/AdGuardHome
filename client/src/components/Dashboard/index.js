import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import 'whatwg-fetch';

import Statistics from './Statistics';
import Counters from './Counters';
import Clients from './Clients';
import QueriedDomains from './QueriedDomains';
import BlockedDomains from './BlockedDomains';

import PageTitle from '../ui/PageTitle';
import Loading from '../ui/Loading';

class Dashboard extends Component {
    componentDidMount() {
        this.props.getStats();
        this.props.getStatsHistory();
        this.props.getTopStats();
    }

    render() {
        const { dashboard } = this.props;
        const dashboardProcessing =
            dashboard.processing ||
            dashboard.processingStats ||
            dashboard.processingStatsHistory ||
            dashboard.processingTopStats;

        const disableButton = <button type="button" className="btn btn-outline-secondary btn-sm mr-2" onClick={() => this.props.disableDns()}>Disable DNS</button>;
        const refreshFullButton = <button type="button" className="btn btn-outline-primary btn-sm" onClick={() => this.props.getStats()}>Refresh statistics</button>;
        const refreshButton = <button type="button" className="btn btn-outline-primary btn-sm card-refresh" onClick={() => this.props.getStats()}></button>;

        return (
            <Fragment>
                <PageTitle title="Dashboard">
                    <div className="page-title__actions">
                        {disableButton}
                        {refreshFullButton}
                    </div>
                </PageTitle>
                {dashboardProcessing && <Loading />}
                {!dashboardProcessing &&
                    <div className="row row-cards">
                        {dashboard.statsHistory &&
                            <div className="col-lg-12">
                                <Statistics
                                    history={dashboard.statsHistory}
                                    refreshButton={refreshButton}
                                />
                            </div>
                        }
                        <div className="col-lg-6">
                            {dashboard.stats &&
                                <Counters
                                    refreshButton={refreshButton}
                                    dnsQueries={dashboard.stats.dns_queries}
                                    blockedFiltering={dashboard.stats.blocked_filtering}
                                    replacedSafebrowsing={dashboard.stats.replaced_safebrowsing}
                                    replacedParental={dashboard.stats.replaced_parental}
                                    replacedSafesearch={dashboard.stats.replaced_safesearch}
                                    avgProcessingTime={dashboard.stats.avg_processing_time}
                                />
                            }
                        </div>
                        {dashboard.topStats &&
                            <Fragment>
                                <div className="col-lg-6">
                                    <Clients
                                        refreshButton={refreshButton}
                                        topClients={dashboard.topStats.top_clients}
                                    />
                                </div>
                                <div className="col-lg-6">
                                    <QueriedDomains
                                        refreshButton={refreshButton}
                                        topQueriedDomains={dashboard.topStats.top_queried_domains}
                                    />
                                </div>
                                <div className="col-lg-6">
                                    <BlockedDomains
                                        refreshButton={refreshButton}
                                        topBlockedDomains={dashboard.topStats.top_blocked_domains}
                                    />
                                </div>
                            </Fragment>
                        }
                    </div>
                }
            </Fragment>
        );
    }
}

Dashboard.propTypes = {
    getStats: PropTypes.func,
    getStatsHistory: PropTypes.func,
    getTopStats: PropTypes.func,
    disableDns: PropTypes.func,
    dashboard: PropTypes.object,
    isCoreRunning: PropTypes.bool,
};

export default Dashboard;
