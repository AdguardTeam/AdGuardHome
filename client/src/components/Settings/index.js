import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';

import Services from './Services';
import StatsConfig from './StatsConfig';
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
        return (
            <div>
                <Trans>no_settings</Trans>
            </div>
        );
    };

    render() {
        const {
            settings,
            services,
            setBlockedServices,
            setStatsConfig,
            stats,
            t,
        } = this.props;
        return (
            <Fragment>
                <PageTitle title={t('general_settings')} />
                {settings.processing && <Loading />}
                {!settings.processing && (
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
                                    setStatsConfig={setStatsConfig}
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
    initSettings: PropTypes.func,
    settings: PropTypes.object,
    settingsList: PropTypes.object,
    toggleSetting: PropTypes.func,
    getStatsConfig: PropTypes.func,
    setStatsConfig: PropTypes.func,
    t: PropTypes.func,
};

export default withNamespaces()(Settings);
