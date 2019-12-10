import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Upstream from './Upstream';
import Access from './Access';
import Rewrites from './Rewrites';
import Config from './Config';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';

class Dns extends Component {
    componentDidMount() {
        this.props.getDnsSettings();
        this.props.getAccessList();
        this.props.getRewritesList();
        this.props.getDnsConfig();
    }

    render() {
        const {
            t,
            dashboard,
            settings,
            access,
            rewrites,
            setAccessList,
            testUpstream,
            setUpstream,
            getRewritesList,
            addRewrite,
            deleteRewrite,
            toggleRewritesModal,
            dnsConfig,
            setDnsConfig,
        } = this.props;

        const isDataLoading = dashboard.processingDnsSettings
            || access.processing
            || rewrites.processing
            || dnsConfig.processingGetConfig;
        const isDataReady = !dashboard.processingDnsSettings
            && !access.processing
            && !rewrites.processing
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
                        <Rewrites
                            rewrites={rewrites}
                            getRewritesList={getRewritesList}
                            addRewrite={addRewrite}
                            deleteRewrite={deleteRewrite}
                            toggleRewritesModal={toggleRewritesModal}
                        />
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
    rewrites: PropTypes.object.isRequired,
    getRewritesList: PropTypes.func.isRequired,
    addRewrite: PropTypes.func.isRequired,
    deleteRewrite: PropTypes.func.isRequired,
    toggleRewritesModal: PropTypes.func.isRequired,
    getDnsSettings: PropTypes.func.isRequired,
    dnsConfig: PropTypes.object.isRequired,
    setDnsConfig: PropTypes.func.isRequired,
    getDnsConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Dns);
