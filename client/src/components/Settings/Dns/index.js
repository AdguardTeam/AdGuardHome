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
        this.props.getAccessList();
        this.props.getDnsConfig();
    }

    render() {
        const {
            t,
            settings,
            access,
            setAccessList,
            testUpstream,
            dnsConfig,
            setDnsConfig,
        } = this.props;

        const isDataLoading = access.processing || dnsConfig.processingGetConfig;

        return (
            <Fragment>
                <PageTitle title={t('dns_settings')} />
                {isDataLoading ?
                    <Loading /> :
                    <Fragment>
                        <Upstream
                            processingTestUpstream={settings.processingTestUpstream}
                            testUpstream={testUpstream}
                            dnsConfig={dnsConfig}
                            setDnsConfig={setDnsConfig}
                        />
                        <Config
                            dnsConfig={dnsConfig}
                            setDnsConfig={setDnsConfig}
                        />
                        <Access access={access} setAccessList={setAccessList} />
                    </Fragment>}
            </Fragment>
        );
    }
}

Dns.propTypes = {
    settings: PropTypes.object.isRequired,
    testUpstream: PropTypes.func.isRequired,
    getAccessList: PropTypes.func.isRequired,
    setAccessList: PropTypes.func.isRequired,
    access: PropTypes.object.isRequired,
    dnsConfig: PropTypes.object.isRequired,
    setDnsConfig: PropTypes.func.isRequired,
    getDnsConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Dns);
