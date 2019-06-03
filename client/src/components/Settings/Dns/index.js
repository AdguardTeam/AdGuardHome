import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Upstream from './Upstream';
import PageTitle from '../../ui/PageTitle';

const Dns = (props) => {
    const { dashboard, settings, t } = props;

    return (
        <Fragment>
            <PageTitle title={t('dns_settings')} />
            <Upstream
                upstreamDns={dashboard.upstreamDns}
                bootstrapDns={dashboard.bootstrapDns}
                allServers={dashboard.allServers}
                setUpstream={props.setUpstream}
                testUpstream={props.testUpstream}
                processingTestUpstream={settings.processingTestUpstream}
                processingSetUpstream={settings.processingSetUpstream}
            />
        </Fragment>
    );
};

Dns.propTypes = {
    dashboard: PropTypes.object.isRequired,
    settings: PropTypes.object.isRequired,
    setUpstream: PropTypes.func.isRequired,
    testUpstream: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Dns);
