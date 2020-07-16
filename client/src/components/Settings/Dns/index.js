import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { useTranslation } from 'react-i18next';

import Upstream from './Upstream';
import Access from './Access';
import Config from './Config';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';
import CacheConfig from './Cache';

const Dns = (props) => {
    const { t } = useTranslation();

    useEffect(() => {
        props.getAccessList();
        props.getDnsConfig();
    }, []);

    const {
        settings,
        access,
        setAccessList,
        dnsConfig,
        setDnsConfig,
    } = props;

    const isDataLoading = access.processing || dnsConfig.processingGetConfig;

    return (
        <>
            <PageTitle title={t('dns_settings')} />
            {isDataLoading
                ? <Loading />
                : <>
                    <Upstream
                        processingTestUpstream={settings.processingTestUpstream}
                        dnsConfig={dnsConfig}
                    />
                    <Config
                        dnsConfig={dnsConfig}
                        setDnsConfig={setDnsConfig}
                    />
                    <CacheConfig
                        dnsConfig={dnsConfig}
                        setDnsConfig={setDnsConfig}
                    />
                    <Access
                        access={access}
                        setAccessList={setAccessList}
                    />
                </>}
        </>
    );
};

Dns.propTypes = {
    settings: PropTypes.object.isRequired,
    getAccessList: PropTypes.func.isRequired,
    setAccessList: PropTypes.func.isRequired,
    access: PropTypes.object.isRequired,
    dnsConfig: PropTypes.object.isRequired,
    setDnsConfig: PropTypes.func.isRequired,
    getDnsConfig: PropTypes.func.isRequired,
};

export default Dns;
