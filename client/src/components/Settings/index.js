import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';

import Upstream from './Upstream';
import Dhcp from './Dhcp';
import Encryption from './Encryption';
import Clients from './Clients';
import AutoClients from './Clients/AutoClients';
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
            settings, dashboard, clients, t,
        } = this.props;
        return (
            <Fragment>
                <PageTitle title={t('settings')} />
                {settings.processing && <Loading />}
                {!settings.processing && (
                    <div className="content">
                        <div className="row">
                            <div className="col-md-12">
                                <Card
                                    title={t('general_settings')}
                                    bodyType="card-body box-body--settings"
                                >
                                    <div className="form">
                                        {this.renderSettings(settings.settingsList)}
                                    </div>
                                </Card>
                                <Upstream
                                    upstreamDns={dashboard.upstreamDns}
                                    bootstrapDns={dashboard.bootstrapDns}
                                    allServers={dashboard.allServers}
                                    setUpstream={this.props.setUpstream}
                                    testUpstream={this.props.testUpstream}
                                    processingTestUpstream={settings.processingTestUpstream}
                                    processingSetUpstream={settings.processingSetUpstream}
                                />
                                {!dashboard.processingTopStats && !dashboard.processingClients && (
                                    <Fragment>
                                        <Clients
                                            clients={dashboard.clients}
                                            topStats={dashboard.topStats}
                                            isModalOpen={clients.isModalOpen}
                                            modalClientName={clients.modalClientName}
                                            modalType={clients.modalType}
                                            addClient={this.props.addClient}
                                            updateClient={this.props.updateClient}
                                            deleteClient={this.props.deleteClient}
                                            toggleClientModal={this.props.toggleClientModal}
                                            processingAdding={clients.processingAdding}
                                            processingDeleting={clients.processingDeleting}
                                            processingUpdating={clients.processingUpdating}
                                        />
                                        <AutoClients
                                            autoClients={dashboard.autoClients}
                                            topStats={dashboard.topStats}
                                        />
                                    </Fragment>
                                )}
                                <Encryption
                                    encryption={this.props.encryption}
                                    setTlsConfig={this.props.setTlsConfig}
                                    validateTlsConfig={this.props.validateTlsConfig}
                                />
                                <Dhcp
                                    dhcp={this.props.dhcp}
                                    toggleDhcp={this.props.toggleDhcp}
                                    getDhcpStatus={this.props.getDhcpStatus}
                                    findActiveDhcp={this.props.findActiveDhcp}
                                    setDhcpConfig={this.props.setDhcpConfig}
                                    addStaticLease={this.props.addStaticLease}
                                    removeStaticLease={this.props.removeStaticLease}
                                    toggleLeaseModal={this.props.toggleLeaseModal}
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
    handleUpstreamChange: PropTypes.func,
    setUpstream: PropTypes.func,
    t: PropTypes.func,
};

export default withNamespaces()(Settings);
