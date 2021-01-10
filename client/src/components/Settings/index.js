import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';

import StatsConfig from './StatsConfig';
import LogsConfig from './LogsConfig';
import FiltersConfig from './FiltersConfig';

import Checkbox from '../ui/Checkbox';
import Loading from '../ui/Loading';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import { getObjectKeysSorted } from '../../helpers/helpers';
import './Settings.css';

const ORDER_KEY = 'order';

const SETTINGS = {
    safebrowsing: {
        enabled: false,
        title: 'use_adguard_browsing_sec',
        subtitle: 'use_adguard_browsing_sec_hint',
        [ORDER_KEY]: 0,
    },
    parental: {
        enabled: false,
        title: 'use_adguard_parental',
        subtitle: 'use_adguard_parental_hint',
        [ORDER_KEY]: 1,
    },
    safesearch: {
        enabled: false,
        title: 'enforce_safe_search',
        subtitle: 'enforce_save_search_hint',
        [ORDER_KEY]: 2,
    },
};

class Settings extends Component {
    componentDidMount() {
        this.props.initSettings(SETTINGS);
        this.props.getStatsConfig();
        this.props.getLogsConfig();
        this.props.getFilteringStatus();
    }

    renderSettings = (settings) => getObjectKeysSorted(settings, ORDER_KEY)
        .map((key) => {
            const setting = settings[key];
            const { enabled } = setting;
            return <Checkbox
                {...setting}
                key={key}
                handleChange={() => this.props.toggleSetting(key, enabled)}
            />;
        });

    render() {
        const {
            settings,
            setStatsConfig,
            resetStats,
            stats,
            queryLogs,
            setLogsConfig,
            clearLogs,
            filtering,
            setFiltersConfig,
            t,
        } = this.props;

        const isDataReady = !settings.processing
            && !stats.processingGetConfig
            && !queryLogs.processingGetConfig;

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
                                            initialValues={{
                                                interval: filtering.interval,
                                                enabled: filtering.enabled,
                                            }}
                                            processing={filtering.processingSetConfig}
                                            setFiltersConfig={setFiltersConfig}
                                        />
                                        {this.renderSettings(settings.settingsList)}
                                    </div>
                                </Card>
                            </div>
                            <div className="col-md-12">
                                <LogsConfig
                                    enabled={queryLogs.enabled}
                                    interval={queryLogs.interval}
                                    anonymize_client_ip={queryLogs.anonymize_client_ip}
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
    t: PropTypes.func.isRequired,
    getLogsConfig: PropTypes.func,
    setLogsConfig: PropTypes.func,
    clearLogs: PropTypes.func,
    stats: PropTypes.shape({
        processingGetConfig: PropTypes.bool,
        interval: PropTypes.number,
        processingSetConfig: PropTypes.bool,
        processingReset: PropTypes.bool,
    }),
    queryLogs: PropTypes.shape({
        enabled: PropTypes.bool,
        interval: PropTypes.number,
        anonymize_client_ip: PropTypes.bool,
        processingSetConfig: PropTypes.bool,
        processingClear: PropTypes.bool,
        processingGetConfig: PropTypes.bool,
    }),
    filtering: PropTypes.shape({
        interval: PropTypes.number,
        enabled: PropTypes.bool,
        processingSetConfig: PropTypes.bool,
    }),
};

export default withTranslation()(Settings);
