import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import Card from '../../../ui/Card';
import Form from './Form';
import { setDnsConfig } from '../../../../actions/dnsConfig';

const Config = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        blocking_mode,
        ratelimit,
        blocking_ipv4,
        blocking_ipv6,
        edns_cs_enabled,
        dnssec_enabled,
        disable_ipv6,
        processingSetConfig,
    } = useSelector((state) => state.dnsConfig, shallowEqual);

    const handleFormSubmit = (values) => {
        dispatch(setDnsConfig(values));
    };

    return (
        <Card
            title={t('dns_config')}
            bodyType="card-body box-body--settings"
            id="dns-config"
        >
            <div className="form">
                <Form
                    initialValues={{
                        ratelimit,
                        blocking_mode,
                        blocking_ipv4,
                        blocking_ipv6,
                        edns_cs_enabled,
                        disable_ipv6,
                        dnssec_enabled,
                    }}
                    onSubmit={handleFormSubmit}
                    processing={processingSetConfig}
                />
            </div>
        </Card>
    );
};

export default Config;
