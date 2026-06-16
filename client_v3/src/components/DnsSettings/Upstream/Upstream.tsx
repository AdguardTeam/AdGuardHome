import { createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
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

        const upstreamDnsFile = dnsConfigState.upstream_dns_file;
        const dnsConfig = {
            fallback_dns,
            bootstrap_dns,
            upstream_mode,
            resolve_clients,
            local_ptr_upstreams,
            use_private_ptr_resolvers,
            upstream_timeout,
            ...(upstreamDnsFile ? null : { upstream_dns }),
        };

        setDnsConfig(dnsConfig);
    };

    const upstreamDns = createMemo(() =>
        dnsConfigState.upstream_dns_file
            ? intl.getMessage('upstream_dns_configured_in_file', { path: dnsConfigState.upstream_dns_file })
            : dnsConfigState.upstream_dns,
    );

    return (
        <div>
            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('upstream_dns')}
            </h2>

            <Form
                initialValues={{
                    upstream_dns: upstreamDns(),
                    fallback_dns: dnsConfigState.fallback_dns,
                    bootstrap_dns: dnsConfigState.bootstrap_dns,
                    upstream_mode: dnsConfigState.upstream_mode,
                    resolve_clients: dnsConfigState.resolve_clients,
                    local_ptr_upstreams: dnsConfigState.local_ptr_upstreams,
                    use_private_ptr_resolvers: dnsConfigState.use_private_ptr_resolvers,
                    upstream_timeout: dnsConfigState.upstream_timeout,
                }}
                onSubmit={handleSubmit}
            />
        </div>
    );
};
