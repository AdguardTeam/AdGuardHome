import cn from 'clsx';

import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Form } from './blocks/Form';

export const ServerConfig = () => {
    const handleFormSubmit = (values: any) => {
        setDnsConfig(values, intl.getMessage('dns_server_configuration_saved_toast'));
    };

    return (
        <div>
            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('dns_config')}
            </h2>

            <Form
                initialValues={{
                    ratelimit: dnsConfigState.ratelimit,
                    ratelimit_subnet_len_ipv4: dnsConfigState.ratelimit_subnet_len_ipv4,
                    ratelimit_subnet_len_ipv6: dnsConfigState.ratelimit_subnet_len_ipv6,
                    ratelimit_whitelist: dnsConfigState.ratelimit_whitelist,
                    blocking_mode: dnsConfigState.blocking_mode,
                    blocking_ipv4: dnsConfigState.blocking_ipv4,
                    blocking_ipv6: dnsConfigState.blocking_ipv6,
                    blocked_response_ttl: dnsConfigState.blocked_response_ttl,
                    edns_cs_enabled: dnsConfigState.edns_cs_enabled,
                    disable_ipv6: dnsConfigState.disable_ipv6,
                    dnssec_enabled: dnsConfigState.dnssec_enabled,
                    edns_cs_use_custom: dnsConfigState.edns_cs_use_custom,
                    edns_cs_custom_ip: dnsConfigState.edns_cs_custom_ip,
                }}
                onSubmit={handleFormSubmit}
                processing={dnsConfigState.processingSetConfig}
            />
        </div>
    );
};
