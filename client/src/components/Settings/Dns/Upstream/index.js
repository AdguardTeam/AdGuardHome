import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import Form from './Form';
import Card from '../../../ui/Card';
import { setDnsConfig } from '../../../../actions/dnsConfig';

const Upstream = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        upstream_dns,
        bootstrap_dns,
        upstream_mode,
    } = useSelector((state) => state.dnsConfig, shallowEqual);

    const upstream_dns_file = useSelector((state) => state.dnsConfig.upstream_dns_file);

    const handleSubmit = (values) => {
        const {
            bootstrap_dns,
            upstream_dns,
            upstream_mode,
        } = values;

        const dnsConfig = {
            bootstrap_dns,
            upstream_mode,
            ...(upstream_dns_file ? null : { upstream_dns }),
        };

        dispatch(setDnsConfig(dnsConfig));
    };

    const upstreamDns = upstream_dns_file ? t('upstream_dns_configured_in_file', { path: upstream_dns_file }) : upstream_dns;

    return <Card
        title={t('upstream_dns')}
        bodyType="card-body box-body--settings"
    >
        <div className="row">
            <div className="col">
                <Form
                    initialValues={{
                        upstream_dns: upstreamDns,
                        bootstrap_dns,
                        upstream_mode,
                    }}
                    onSubmit={handleSubmit}
                />
            </div>
        </div>
    </Card>;
};

export default Upstream;
