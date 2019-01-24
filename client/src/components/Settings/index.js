import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';
import Upstream from './Upstream';
import Dhcp from './Dhcp';
import Encryption from './Encryption';
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
        this.props.getDhcpStatus();
        this.props.getDhcpInterfaces();
        this.props.getTlsStatus();
    }

    handleUpstreamChange = (value) => {
        this.props.handleUpstreamChange({ upstreamDns: value });
    };

    handleUpstreamSubmit = () => {
        this.props.setUpstream(this.props.dashboard.upstreamDns);
    };

    handleUpstreamTest = () => {
        if (this.props.dashboard.upstreamDns.length > 0) {
            this.props.testUpstream(this.props.dashboard.upstreamDns);
        } else {
            this.props.addErrorToast({ error: this.props.t('no_servers_specified') });
        }
    };

    renderSettings = (settings) => {
        if (Object.keys(settings).length > 0) {
            return Object.keys(settings).map((key) => {
                const setting = settings[key];
                const { enabled } = setting;
                return (<Checkbox
                    key={key}
                    {...settings[key]}
                    handleChange={() => this.props.toggleSetting(key, enabled)}
                    />);
            });
        }
        return (
            <div><Trans>no_settings</Trans></div>
        );
    }

    render() {
        const { settings, t } = this.props;
        const { upstreamDns } = this.props.dashboard;
        return (
            <Fragment>
                <PageTitle title={ t('settings') } />
                {settings.processing && <Loading />}
                {!settings.processing &&
                    <div className="content">
                        <div className="row">
                            <div className="col-md-12">
                                <Card title={ t('general_settings') } bodyType="card-body box-body--settings">
                                    <div className="form">
                                        {this.renderSettings(settings.settingsList)}
                                    </div>
                                </Card>
                                <Upstream
                                    upstreamDns={upstreamDns}
                                    processingTestUpstream={settings.processingTestUpstream}
                                    handleUpstreamChange={this.handleUpstreamChange}
                                    handleUpstreamSubmit={this.handleUpstreamSubmit}
                                    handleUpstreamTest={this.handleUpstreamTest}
                                />
                                <Encryption
                                    encryption={this.props.encryption}
                                    setTlsConfig={this.props.setTlsConfig}
                                />
                                <Dhcp
                                    dhcp={this.props.dhcp}
                                    toggleDhcp={this.props.toggleDhcp}
                                    getDhcpStatus={this.props.getDhcpStatus}
                                    findActiveDhcp={this.props.findActiveDhcp}
                                    setDhcpConfig={this.props.setDhcpConfig}
                                />
                            </div>
                        </div>
                    </div>
                }
            </Fragment>
        );
    }
}

Settings.propTypes = {
    initSettings: PropTypes.func,
    settings: PropTypes.object,
    settingsList: PropTypes.object,
    toggleSetting: PropTypes.func,
    handleUpstreamChange: PropTypes.func,
    setUpstream: PropTypes.func,
    upstream: PropTypes.string,
    t: PropTypes.func,
};

export default withNamespaces()(Settings);
