import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { Trans, useTranslation } from 'react-i18next';

import Statistics from './Statistics';
import Counters from './Counters';
import Clients from './Clients';
import QueriedDomains from './QueriedDomains';
import BlockedDomains from './BlockedDomains';

import PageTitle from '../ui/PageTitle';
import Loading from '../ui/Loading';
import { BLOCK_ACTIONS } from '../../helpers/constants';
import './Dashboard.css';

const Dashboard = ({
    getAccessList,
    getStats,
    getStatsConfig,
    dashboard,
    toggleProtection,
    toggleClientBlock,
    stats,
    access,
}) => {
    const { t } = useTranslation();

    const getAllStats = () => {
        getAccessList();
        getStats();
        getStatsConfig();
    };

    useEffect(() => {
        getAllStats();
    }, []);

    const getToggleFilteringButton = () => {
        const { protectionEnabled, processingProtection } = dashboard;
        const buttonText = protectionEnabled ? 'disable_protection' : 'enable_protection';
        const buttonClass = protectionEnabled ? 'btn-gray' : 'btn-success';

        return <button
                type="button"
                className={`btn btn-sm mr-2 ${buttonClass}`}
                onClick={() => toggleProtection(protectionEnabled)}
                disabled={processingProtection}
        >
            <Trans>{buttonText}</Trans>
        </button>;
    };

    const toggleClientStatus = (type, ip) => {
        const confirmMessage = type === BLOCK_ACTIONS.BLOCK ? 'client_confirm_block' : 'client_confirm_unblock';

        if (window.confirm(t(confirmMessage, { ip }))) {
            toggleClientBlock(type, ip);
        }
    };

    const refreshButton = <button
            type="button"
            className="btn btn-icon btn-outline-primary btn-sm"
            onClick={() => getAllStats()}
    >
        <svg className="icons">
            <use xlinkHref="#refresh" />
        </svg>
    </button>;

    const subtitle = stats.interval === 1
        ? t('for_last_24_hours')
        : t('for_last_days', { count: stats.interval });

    const refreshFullButton = <button
            type="button"
            className="btn btn-outline-primary btn-sm"
            onClick={() => getAllStats()}
    >
        <Trans>refresh_statics</Trans>
    </button>;

    const statsProcessing = stats.processingStats
            || stats.processingGetConfig
            || access.processing;

    return <>
        <PageTitle title={t('dashboard')}>
            <div className="page-title__actions">
                {getToggleFilteringButton()}
                {refreshFullButton}
            </div>
        </PageTitle>
        {statsProcessing && <Loading />}
        {!statsProcessing && <div className="row row-cards">
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
                        toggleClientStatus={toggleClientStatus}
                        processingAccessSet={access.processingSet}
                        disallowedClients={access.disallowed_clients}
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
        </div>}
    </>;
};

Dashboard.propTypes = {
    dashboard: PropTypes.object.isRequired,
    stats: PropTypes.object.isRequired,
    access: PropTypes.object.isRequired,
    getStats: PropTypes.func.isRequired,
    getStatsConfig: PropTypes.func.isRequired,
    toggleProtection: PropTypes.func.isRequired,
    getClients: PropTypes.func.isRequired,
    toggleClientBlock: PropTypes.func.isRequired,
    getAccessList: PropTypes.func.isRequired,
};

export default Dashboard;
