import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Upstream from './Upstream';
import Access from './Access';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';

class Dns extends Component {
    componentDidMount() {
        this.props.getAccessList();
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
        } = this.props;

        return (
            <Fragment>
                <PageTitle title={t('dns_settings')} />
                {(dashboard.processing || access.processing) && <Loading />}
                {!dashboard.processing && !access.processing && (
                    <Fragment>
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
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Dns);
