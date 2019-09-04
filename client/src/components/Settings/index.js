import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Services from './Services';
import StatsConfig from './StatsConfig';
import LogsConfig from './LogsConfig';
import Checkbox from '../ui/Checkbox';
import Loading from '../ui/Loading';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';

import './Settings.css';

class Settings extends Component {
    settings = {
        filtering: {
            enabled: false,
            title: 'block_domain_use_filters_and_hosts',
            subtitle: 'filters_block_toggle_hint',
        },
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
    }

    renderSettings = (settings) => {
        if (Object.keys(settings).length > 0) {
            return Object.keys(settings).map((key) => {
                const setting = settings[key];
                const { enabled } = setting;
                return (
                    <Checkbox
                        key={key}
                        {...settings[key]}
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
            setLogsConfig,
            clearLogs,
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
                                        {this.renderSettings(settings.settingsList)}
                                    </div>
                                </Card>
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
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Settings);
