import React from 'react';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';

import { setDnsConfig } from 'panel/actions/dnsConfig';
import { RootState } from 'panel/initialState';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Form } from './blocks/Form';

export const ServerConfig = () => {
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
        <div>
            <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('dns_config')}
            </h2>

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
    );
};
