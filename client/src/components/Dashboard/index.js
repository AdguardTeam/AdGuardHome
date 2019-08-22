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
        this.props.getStatsConfig();
        this.props.getClients();
    };

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
    };

    render() {
        const { dashboard, stats, t } = this.props;
        const dashboardProcessing =
            dashboard.processing ||
            dashboard.processingClients ||
            stats.processingStats ||
            stats.processingGetConfig;

        const subtitle =
            stats.interval === 1
                ? t('for_last_24_hours')
                : t('for_last_days', { value: stats.interval });

        const refreshFullButton = (
            <button
                type="button"
                className="btn btn-outline-primary btn-sm"
                onClick={() => this.getAllStats()}
            >
                <Trans>refresh_statics</Trans>
            </button>
        );

        const refreshButton = (
            <button
                type="button"
                className="btn btn-icon btn-outline-primary btn-sm"
                onClick={() => this.getAllStats()}
            >
                <svg className="icons">
                    <use xlinkHref="#refresh" />
                </svg>
            </button>
        );

        return (
            <Fragment>
                <PageTitle title={t('dashboard')}>
                    <div className="page-title__actions">
                        {this.getToggleFilteringButton()}
                        {refreshFullButton}
                    </div>
                </PageTitle>
                {dashboardProcessing && <Loading />}
                {!dashboardProcessing && (
                    <div className="row row-cards">
                        <div className="col-lg-12">
                            <Statistics
                                interval={stats.interval}
                                dnsQueries={stats.dnsQueries}
                                blockedFiltering={stats.blockedFiltering}
                                replacedSafebrowsing={stats.replacedSafebrowsing}
                                replacedParental={stats.replacedParental}
                                numDnsQueries={stats.numDnsQueries}
                                numBlockedFiltering={stats.numBlockedFiltering}
                                numReplacedSafebrowsing={stats.numReplacedSafebrowsing}
                                numReplacedParental={stats.numReplacedParental}
                                refreshButton={refreshButton}
                            />
                        </div>
                        <div className="col-lg-6">
                            <Counters
                                subtitle={subtitle}
                                interval={stats.interval}
                                dnsQueries={stats.numDnsQueries}
                                blockedFiltering={stats.numBlockedFiltering}
                                replacedSafebrowsing={stats.numReplacedSafebrowsing}
                                replacedParental={stats.numReplacedParental}
                                replacedSafesearch={stats.numReplacedSafesearch}
                                avgProcessingTime={stats.avgProcessingTime}
                                refreshButton={refreshButton}
                            />
                        </div>
                        <div className="col-lg-6">
                            <Clients
                                subtitle={subtitle}
                                dnsQueries={stats.numDnsQueries}
                                topClients={stats.topClients}
                                clients={dashboard.clients}
                                autoClients={dashboard.autoClients}
                                refreshButton={refreshButton}
                            />
                        </div>
                        <div className="col-lg-6">
                            <QueriedDomains
                                subtitle={subtitle}
                                dnsQueries={stats.numDnsQueries}
                                topQueriedDomains={stats.topQueriedDomains}
                                refreshButton={refreshButton}
                            />
                        </div>
                        <div className="col-lg-6">
                            <BlockedDomains
                                subtitle={subtitle}
                                topBlockedDomains={stats.topBlockedDomains}
                                blockedFiltering={stats.numBlockedFiltering}
                                replacedSafebrowsing={stats.numReplacedSafebrowsing}
                                replacedParental={stats.numReplacedParental}
                                refreshButton={refreshButton}
                            />
                        </div>
                    </div>
                )}
            </Fragment>
        );
    }
}

Dashboard.propTypes = {
    dashboard: PropTypes.object.isRequired,
    stats: PropTypes.object.isRequired,
    getStats: PropTypes.func.isRequired,
    getStatsConfig: PropTypes.func.isRequired,
    toggleProtection: PropTypes.func.isRequired,
    getClients: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Dashboard);
