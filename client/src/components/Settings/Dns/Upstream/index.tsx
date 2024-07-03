import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import Form from './Form';

import Card from '../../../ui/Card';
import { setDnsConfig } from '../../../../actions/dnsConfig';
import { RootState } from '../../../../initialState';

const Upstream = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        upstream_dns,
        fallback_dns,
        bootstrap_dns,
        upstream_mode,
        resolve_clients,
        local_ptr_upstreams,
        use_private_ptr_resolvers,
    } = useSelector((state: RootState) => state.dnsConfig, shallowEqual);

    const upstream_dns_file = useSelector((state: RootState) => state.dnsConfig.upstream_dns_file);

    const handleSubmit = (values: any) => {
        const {
            fallback_dns,
            bootstrap_dns,
            upstream_dns,
            upstream_mode,
            resolve_clients,
            local_ptr_upstreams,
            use_private_ptr_resolvers,
        } = values;

        const dnsConfig = {
            fallback_dns,
            bootstrap_dns,
            upstream_mode,
            resolve_clients,
            local_ptr_upstreams,
            use_private_ptr_resolvers,
            ...(upstream_dns_file ? null : { upstream_dns }),
        };

        dispatch(setDnsConfig(dnsConfig));
    };

    const upstreamDns = upstream_dns_file
        ? t('upstream_dns_configured_in_file', { path: upstream_dns_file })
        : upstream_dns;

    return (
        <Card title={t('upstream_dns')} bodyType="card-body box-body--settings">
            <div className="row">
                <div className="col">
                    <Form
                        initialValues={{
                            upstream_dns: upstreamDns,
                            fallback_dns,
                            bootstrap_dns,
                            upstream_mode,
                            resolve_clients,
                            local_ptr_upstreams,
                            use_private_ptr_resolvers,
                        }}
                        onSubmit={handleSubmit}
                    />
                </div>
            </div>
        </Card>
    );
};

export default Upstream;
