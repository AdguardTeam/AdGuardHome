import cn from 'clsx';

import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import { replaceEmptyStringsWithZeroes, replaceZeroWithEmptyString } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { Form } from './Form';

export const Cache = () => {
    const handleFormSubmit = (values: any) => {
        const completedFields = replaceEmptyStringsWithZeroes(values);
        setDnsConfig(completedFields, intl.getMessage('dns_cache_configuration_saved_toast'));
    };

    return (
        <div>
            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('dns_cache_config')}
            </h2>

            <Form
                initialValues={{
                    cache_enabled: dnsConfigState.cache_enabled,
                    cache_size: replaceZeroWithEmptyString(dnsConfigState.cache_size),
                    cache_ttl_max: replaceZeroWithEmptyString(dnsConfigState.cache_ttl_max),
                    cache_ttl_min: replaceZeroWithEmptyString(dnsConfigState.cache_ttl_min),
                    cache_optimistic: dnsConfigState.cache_optimistic,
                }}
                onSubmit={handleFormSubmit}
            />
        </div>
    );
};
