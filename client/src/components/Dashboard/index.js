import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { HashLink as Link } from 'react-router-hash-link';
import { Trans, useTranslation } from 'react-i18next';
import classNames from 'classnames';

import Statistics from './Statistics';
import Counters from './Counters';
import Clients from './Clients';
import QueriedDomains from './QueriedDomains';
import BlockedDomains from './BlockedDomains';
import {
    DISABLE_PROTECTION_TIMINGS,
    ONE_SECOND_IN_MS,
    SETTINGS_URLS,
    TIME_UNITS,
} from '../../helpers/constants';
import {
    msToSeconds,
    msToMinutes,
    msToHours,
    msToDays,
} from '../../helpers/helpers';

import PageTitle from '../ui/PageTitle';
import Loading from '../ui/Loading';
import './Dashboard.css';
import Dropdown from '../ui/Dropdown';
import UpstreamResponses from './UpstreamResponses';
import UpstreamAvgTime from './UpstreamAvgTime';

const Dashboard = ({
    getAccessList,
    getStats,
    getStatsConfig,
    dashboard,
    dashboard: { protectionEnabled, processingProtection, protectionDisabledDuration },
    toggleProtection,
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
    const getSubtitle = () => {
        if (!stats.enabled) {
            return t('stats_disabled_short');
        }

        const msIn7Days = 604800000;

        if (stats.timeUnits === TIME_UNITS.HOURS && stats.interval === msIn7Days) {
            return t('for_last_days', { count: msToDays(stats.interval) });
        }

        return stats.timeUnits === TIME_UNITS.HOURS
            ? t('for_last_hours', { count: msToHours(stats.interval) })
            : t('for_last_days', { count: msToDays(stats.interval) });
    };

    const buttonClass = classNames('btn btn-sm dashboard-protection-button', {
        'btn-gray': protectionEnabled,
        'btn-success': !protectionEnabled,
    });

    const refreshButton = <button
            type="button"
            className="btn btn-icon btn-outline-primary btn-sm"
            title={t('refresh_btn')}
            onClick={() => getAllStats()}
    >
        <svg className="icons icon12">
            <use xlinkHref="#refresh" />
        </svg>
    </button>;

    const statsProcessing = stats.processingStats
            || stats.processingGetConfig
            || access.processing;

    const subtitle = getSubtitle();

    const DISABLE_PROTECTION_ITEMS = [
        {
            text: t('disable_for_seconds', { count: msToSeconds(DISABLE_PROTECTION_TIMINGS.HALF_MINUTE) }),
            disableTime: DISABLE_PROTECTION_TIMINGS.HALF_MINUTE,
        },
        {
            text: t('disable_for_minutes', { count: msToMinutes(DISABLE_PROTECTION_TIMINGS.MINUTE) }),
            disableTime: DISABLE_PROTECTION_TIMINGS.MINUTE,
        },
        {
            text: t('disable_for_minutes', { count: msToMinutes(DISABLE_PROTECTION_TIMINGS.TEN_MINUTES) }),
            disableTime: DISABLE_PROTECTION_TIMINGS.TEN_MINUTES,
        },
        {
            text: t('disable_for_hours', { count: msToHours(DISABLE_PROTECTION_TIMINGS.HOUR) }),
            disableTime: DISABLE_PROTECTION_TIMINGS.HOUR,
        },
        {
            text: t('disable_until_tomorrow'),
            disableTime: DISABLE_PROTECTION_TIMINGS.TOMORROW,
        },
    ];

    const getDisableProtectionItems = () => (
        Object.values(DISABLE_PROTECTION_ITEMS)
            .map((item, index) => (
                <div
                    key={`disable_timings_${index}`}
                    className="dropdown-item"
                    onClick={() => {
                        toggleProtection(protectionEnabled, item.disableTime - ONE_SECOND_IN_MS);
                    }}
                >
                    {item.text}
                </div>
            ))
    );

    const getRemaningTimeText = (milliseconds) => {
        if (!milliseconds) {
            return '';
        }

        const date = new Date(milliseconds);
        const hh = date.getUTCHours();
        const mm = `0${date.getUTCMinutes()}`.slice(-2);
        const ss = `0${date.getUTCSeconds()}`.slice(-2);
        const formattedHH = `0${hh}`.slice(-2);

        return hh ? `${formattedHH}:${mm}:${ss}` : `${mm}:${ss}`;
    };

    const getProtectionBtnText = (status) => (status ? t('disable_protection') : t('enable_protection'));

    return <>
        <PageTitle title={t('dashboard')} containerClass="page-title--dashboard">
            <div className="page-title__protection">
                <button
                    type="button"
                    className={buttonClass}
                    onClick={() => {
                        toggleProtection(protectionEnabled);
                    }}
                    disabled={processingProtection}
                >
                    {protectionDisabledDuration
                        ? `${t('enable_protection_timer')} ${getRemaningTimeText(protectionDisabledDuration)}`
                        : getProtectionBtnText(protectionEnabled)
                    }
                </button>

                {protectionEnabled && <Dropdown
                    label=""
                    baseClassName="dropdown-protection"
                    icon="arrow-down"
                    controlClassName="dropdown-protection__toggle"
                    menuClassName="dropdown-menu dropdown-menu-arrow dropdown-menu--protection"
                >
                    {getDisableProtectionItems()}
                </Dropdown>}
            </div>
            <button
                type="button"
                className="btn btn-outline-primary btn-sm"
                onClick={getAllStats}
            >
                <Trans>refresh_statics</Trans>
            </button>
        </PageTitle>
        {statsProcessing && <Loading />}
        {!statsProcessing && <div className="row row-cards dashboard">
            <div className="col-lg-12">
                {stats.interval === 0 && (
                    <div className="alert alert-warning" role="alert">
                        <Trans components={[
                            <Link
                                to={`${SETTINGS_URLS.settings}#stats-config`}
                                key="0"
                            >
                                link
                            </Link>,
                        ]}>
                            stats_disabled
                        </Trans>
                    </div>
                )}
                <Statistics
                    interval={msToDays(stats.interval)}
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
                    replacedSafesearch={stats.numReplacedSafesearch}
                    replacedParental={stats.numReplacedParental}
                    refreshButton={refreshButton}
                />
            </div>
            <div className="col-lg-6">
                <UpstreamResponses
                    subtitle={subtitle}
                    topUpstreamsResponses={stats.topUpstreamsResponses}
                    refreshButton={refreshButton}
                />
            </div>
            <div className="col-lg-6">
                <UpstreamAvgTime
                    subtitle={subtitle}
                    topUpstreamsAvgTime={stats.topUpstreamsAvgTime}
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
    getAccessList: PropTypes.func.isRequired,
};

export default Dashboard;
