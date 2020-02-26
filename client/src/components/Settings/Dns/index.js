import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Upstream from './Upstream';
import Access from './Access';
import Config from './Config';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';

class Dns extends Component {
    componentDidMount() {
        this.props.getDnsSettings();
        this.props.getAccessList();
        this.props.getDnsConfig();
    }

    render() {
        const {
            t,
            dashboard,
            settings,
            access,
            setAccessList,
            testUpstream,
            setUpstream,
            dnsConfig,
            setDnsConfig,
        } = this.props;

        const isDataLoading = dashboard.processingDnsSettings
            || access.processing
            || dnsConfig.processingGetConfig;
        const isDataReady = !dashboard.processingDnsSettings
            && !access.processing
            && !dnsConfig.processingGetConfig;

        return (
            <Fragment>
                <PageTitle title={t('dns_settings')} />
                {isDataLoading && <Loading />}
                {isDataReady && (
                    <Fragment>
                        <Config
                            dnsConfig={dnsConfig}
                            setDnsConfig={setDnsConfig}
                        />
                        <Upstream
                            upstreamDns={dashboard.upstreamDns}
                            bootstrapDns={dashboard.bootstrapDns}
                            allServers={dashboard.allServers}
                            processingTestUpstream={settings.processingTestUpstream}
                            processingSetUpstream={settings.processingSetUpstream}
                            setUpstream={setUpstream}
                            testUpstream={testUpstream}
                        />
                        <Access access={access} setAccessList={setAccessList} />
                    </Fragment>
                )}
            </Fragment>
        );
    }
}

Dns.propTypes = {
    dashboard: PropTypes.object.isRequired,
    settings: PropTypes.object.isRequired,
    setUpstream: PropTypes.func.isRequired,
    testUpstream: PropTypes.func.isRequired,
    getAccessList: PropTypes.func.isRequired,
    setAccessList: PropTypes.func.isRequired,
    access: PropTypes.object.isRequired,
    getDnsSettings: PropTypes.func.isRequired,
    dnsConfig: PropTypes.object.isRequired,
    setDnsConfig: PropTypes.func.isRequired,
    getDnsConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Dns);
