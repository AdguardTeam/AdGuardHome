import React from 'react';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { setDnsConfig } from 'panel/actions/dnsConfig';
import { RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';

import { Form } from './blocks/Form';

type UpstreamFormData = {
    upstream_dns: string;
    upstream_mode: string;
    fallback_dns: string;
    bootstrap_dns: string;
    local_ptr_upstreams: string;
    use_private_ptr_resolvers: boolean;
    resolve_clients: boolean;
    upstream_timeout: number;
};

export const Upstream = () => {
    const dispatch = useDispatch();
    const {
        upstream_dns,
        fallback_dns,
        bootstrap_dns,
        upstream_mode,
        resolve_clients,
        local_ptr_upstreams,
        use_private_ptr_resolvers,
        upstream_timeout,
    } = useSelector((state: RootState) => state.dnsConfig, shallowEqual);

    const upstream_dns_file = useSelector((state: RootState) => state.dnsConfig.upstream_dns_file);

    const handleSubmit = (values: UpstreamFormData) => {
        const {
            fallback_dns,
            bootstrap_dns,
            upstream_dns,
            upstream_mode,
            resolve_clients,
            local_ptr_upstreams,
            use_private_ptr_resolvers,
            upstream_timeout,
        } = values;

        const dnsConfig = {
            fallback_dns,
            bootstrap_dns,
            upstream_mode,
            resolve_clients,
            local_ptr_upstreams,
            use_private_ptr_resolvers,
            upstream_timeout,
            ...(upstream_dns_file ? null : { upstream_dns }),
        };

        dispatch(setDnsConfig(dnsConfig));
    };

    const upstreamDns = upstream_dns_file
        ? intl.getMessage('upstream_dns_configured_in_file', { path: upstream_dns_file })
        : upstream_dns;

    return (
        <div>
            <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('upstream_dns')}
            </h2>

            <Form
                initialValues={{
                    upstream_dns: upstreamDns,
                    fallback_dns,
                    bootstrap_dns,
                    upstream_mode,
                    resolve_clients,
                    local_ptr_upstreams,
                    use_private_ptr_resolvers,
                    upstream_timeout,
                }}
                onSubmit={handleSubmit}
            />
        </div>
    );
};
