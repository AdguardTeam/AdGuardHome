import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import Card from '../../../ui/Card';

import Form from './Form';
import { setDnsConfig } from '../../../../actions/dnsConfig';
import { RootState } from '../../../../initialState';

const Config = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        blocking_mode,
        ratelimit,
        ratelimit_subnet_len_ipv4,
        ratelimit_subnet_len_ipv6,
        ratelimit_whitelist,
        blocking_ipv4,
        blocking_ipv6,
        blocked_response_ttl,
        edns_cs_enabled,
        edns_cs_use_custom,
        edns_cs_custom_ip,
        dnssec_enabled,
        disable_ipv6,
        processingSetConfig,
    } = useSelector((state: RootState) => state.dnsConfig, shallowEqual);

    const handleFormSubmit = (values: any) => {
        dispatch(setDnsConfig(values));
    };

    return (
        <Card title={t('dns_config')} bodyType="card-body box-body--settings" id="dns-config">
            <div className="form">
                <Form
                    initialValues={{
                        ratelimit,
                        ratelimit_subnet_len_ipv4,
                        ratelimit_subnet_len_ipv6,
                        ratelimit_whitelist,
                        blocking_mode,
                        blocking_ipv4,
                        blocking_ipv6,
                        blocked_response_ttl,
                        edns_cs_enabled,
                        disable_ipv6,
                        dnssec_enabled,
                        edns_cs_use_custom,
                        edns_cs_custom_ip,
                    }}
                    onSubmit={handleFormSubmit}
                    processing={processingSetConfig}
                />
            </div>
        </Card>
    );
};

export default Config;
