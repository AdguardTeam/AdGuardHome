import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Statistics from './Statistics';
import Counters from './Counters';
import Clients from './Clients';
import QueriedDomains from './QueriedDomains';
import BlockedDomains from './BlockedDomains';

import PageTitle from '../ui/PageTitle';
import Loading from '../ui/Loading';
import './Dashboard.css';

class Dashboard extends Component {
    componentDidMount() {
        this.getAllStats();
    }

    getAllStats = () => {
        this.props.getStats();
        this.props.getStatsHistory();
        this.props.getTopStats();
    }

    getToggleFilteringButton = () => {
        const { protectionEnabled, processingProtection } = this.props.dashboard;
        const buttonText = protectionEnabled ? 'disable_protection' : 'enable_protection';
        const buttonClass = protectionEnabled ? 'btn-gray' : 'btn-success';

        return (
            <button
                type="button"
                className={`btn btn-sm mr-2 ${buttonClass}`}
                onClick={() => this.props.toggleProtection(protectionEnabled)}
                disabled={processingProtection}
            >
                <Trans>{buttonText}</Trans>
            </button>
        );
    }

    render() {
        const { dashboard, t } = this.props;
        const dashboardProcessing =
            dashboard.processing ||
            dashboard.processingStats ||
            dashboard.processingStatsHistory ||
            dashboard.processingClients ||
            dashboard.processingTopStats;

        const refreshFullButton = <button type="button" className="btn btn-outline-primary btn-sm" onClick={() => this.getAllStats()}><Trans>refresh_statics</Trans></button>;
        const refreshButton = <button type="button" className="btn btn-outline-primary btn-sm card-refresh" onClick={() => this.getAllStats()} />;

        return (
            <Fragment>
                <PageTitle title={ t('dashboard') }>
                    <div className="page-title__actions">
                        {this.getToggleFilteringButton()}
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
                                    dnsQueries={dashboard.stats.dns_queries}
                                    blockedFiltering={dashboard.stats.blocked_filtering}
                                    replacedSafebrowsing={dashboard.stats.replaced_safebrowsing}
                                    replacedParental={dashboard.stats.replaced_parental}
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
                                        dnsQueries={dashboard.stats.dns_queries}
                                        refreshButton={refreshButton}
                                        topClients={dashboard.topStats.top_clients}
                                        clients={dashboard.clients}
                                    />
                                </div>
                                <div className="col-lg-6">
                                    <QueriedDomains
                                        dnsQueries={dashboard.stats.dns_queries}
                                        refreshButton={refreshButton}
                                        topQueriedDomains={dashboard.topStats.top_queried_domains}
                                    />
                                </div>
                                <div className="col-lg-6">
                                    <BlockedDomains
                                        refreshButton={refreshButton}
                                        topBlockedDomains={dashboard.topStats.top_blocked_domains}
                                        blockedFiltering={dashboard.stats.blocked_filtering}
                                        replacedSafebrowsing={dashboard.stats.replaced_safebrowsing}
                                        replacedParental={dashboard.stats.replaced_parental}
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
    dashboard: PropTypes.object,
    isCoreRunning: PropTypes.bool,
    getFiltering: PropTypes.func,
    toggleProtection: PropTypes.func,
    processingProtection: PropTypes.bool,
    t: PropTypes.func,
};

export default withNamespaces()(Dashboard);
