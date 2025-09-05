import React from 'react';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';

import { setDnsConfig } from 'panel/actions/dnsConfig';
import { replaceEmptyStringsWithZeroes, replaceZeroWithEmptyString } from 'panel/helpers/helpers';
import { RootState } from 'panel/initialState';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { Form } from './Form';

export const Cache = () => {
    const dispatch = useDispatch();
    const { cache_enabled, cache_size, cache_ttl_max, cache_ttl_min, cache_optimistic } = useSelector(
        (state: RootState) => state.dnsConfig,
        shallowEqual,
    );

    const handleFormSubmit = (values: any) => {
        const completedFields = replaceEmptyStringsWithZeroes(values);
        dispatch(setDnsConfig(completedFields));
    };

    return (
        <div>
            <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('dns_cache_config')}
            </h2>

            <Form
                initialValues={{
                    cache_enabled,
                    cache_size: replaceZeroWithEmptyString(cache_size),
                    cache_ttl_max: replaceZeroWithEmptyString(cache_ttl_max),
                    cache_ttl_min: replaceZeroWithEmptyString(cache_ttl_min),
                    cache_optimistic,
                }}
                onSubmit={handleFormSubmit}
            />
        </div>
    );
};
