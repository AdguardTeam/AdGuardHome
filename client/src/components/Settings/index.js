import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Services from './Services';
import StatsConfig from './StatsConfig';
import LogsConfig from './LogsConfig';
import FiltersConfig from './FiltersConfig';
import DnsConfig from './DnsConfig';

import Checkbox from '../ui/Checkbox';
import Loading from '../ui/Loading';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';

import './Settings.css';

class Settings extends Component {
    settings = {
        safebrowsing: {
            enabled: false,
            title: 'use_adguard_browsing_sec',
            subtitle: 'use_adguard_browsing_sec_hint',
        },
        parental: {
            enabled: false,
            title: 'use_adguard_parental',
            subtitle: 'use_adguard_parental_hint',
        },
        safesearch: {
            enabled: false,
            title: 'enforce_safe_search',
            subtitle: 'enforce_save_search_hint',
        },
    };

    componentDidMount() {
        this.props.initSettings(this.settings);
        this.props.getBlockedServices();
        this.props.getStatsConfig();
        this.props.getLogsConfig();
        this.props.getFilteringStatus();
        this.props.getDnsConfig();
    }

    renderSettings = (settings) => {
        const settingsKeys = Object.keys(settings);

        if (settingsKeys.length > 0) {
            return settingsKeys.map((key) => {
                const setting = settings[key];
                const { enabled } = setting;
                return (
                    <Checkbox
                        {...settings[key]}
                        key={key}
                        handleChange={() => this.props.toggleSetting(key, enabled)}
                    />
                );
            });
        }
        return '';
    };

    render() {
        const {
            settings,
            services,
            setBlockedServices,
            setStatsConfig,
            resetStats,
            stats,
            queryLogs,
            dnsConfig,
            setLogsConfig,
            clearLogs,
            filtering,
            setFiltersConfig,
            setDnsConfig,
            t,
        } = this.props;

        const isDataReady =
            !settings.processing &&
            !services.processing &&
            !stats.processingGetConfig &&
            !queryLogs.processingGetConfig;

        return (
            <Fragment>
                <PageTitle title={t('general_settings')} />
                {!isDataReady && <Loading />}
                {isDataReady && (
                    <div className="content">
                        <div className="row">
                            <div className="col-md-12">
                                <Card bodyType="card-body box-body--settings">
                                    <div className="form">
                                        <FiltersConfig
                                            interval={filtering.interval}
                                            enabled={filtering.enabled}
                                            processing={filtering.processingSetConfig}
                                            setFiltersConfig={setFiltersConfig}
                                        />
                                        {this.renderSettings(settings.settingsList)}
                                    </div>
                                </Card>
                            </div>
                            <div className="col-md-12">
                                <DnsConfig
                                    dnsConfig={dnsConfig}
                                    setDnsConfig={setDnsConfig}
                                />
                            </div>
                            <div className="col-md-12">
                                <LogsConfig
                                    enabled={queryLogs.enabled}
                                    interval={queryLogs.interval}
                                    processing={queryLogs.processingSetConfig}
                                    processingClear={queryLogs.processingClear}
                                    setLogsConfig={setLogsConfig}
                                    clearLogs={clearLogs}
                                />
                            </div>
                            <div className="col-md-12">
                                <StatsConfig
                                    interval={stats.interval}
                                    processing={stats.processingSetConfig}
                                    processingReset={stats.processingReset}
                                    setStatsConfig={setStatsConfig}
                                    resetStats={resetStats}
                                />
                            </div>
                            <div className="col-md-12">
                                <Services
                                    services={services}
                                    setBlockedServices={setBlockedServices}
                                />
                            </div>
                        </div>
                    </div>
                )}
            </Fragment>
        );
    }
}

Settings.propTypes = {
    initSettings: PropTypes.func.isRequired,
    settings: PropTypes.object.isRequired,
    toggleSetting: PropTypes.func.isRequired,
    getStatsConfig: PropTypes.func.isRequired,
    setStatsConfig: PropTypes.func.isRequired,
    resetStats: PropTypes.func.isRequired,
    setFiltersConfig: PropTypes.func.isRequired,
    getFilteringStatus: PropTypes.func.isRequired,
    getDnsConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Settings);
