import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import Card from '../../../ui/Card';
import Form from './Form';
import { setDnsConfig } from '../../../../actions/dnsConfig';
import { replaceEmptyStringsWithZeroes, replaceZeroWithEmptyString } from '../../../../helpers/helpers';

const CacheConfig = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        cache_size, cache_ttl_max, cache_ttl_min,
    } = useSelector((state) => state.dnsConfig, shallowEqual);

    const handleFormSubmit = (values) => {
        const completedFields = replaceEmptyStringsWithZeroes(values);
        dispatch(setDnsConfig(completedFields));
    };

    return (
        <Card
            title={t('dns_cache_config')}
            subtitle={t('dns_cache_config_desc')}
            bodyType="card-body box-body--settings"
            id="dns-config"
        >
            <div className="form">
                <Form
                    initialValues={{
                        cache_size: replaceZeroWithEmptyString(cache_size),
                        cache_ttl_max: replaceZeroWithEmptyString(cache_ttl_max),
                        cache_ttl_min: replaceZeroWithEmptyString(cache_ttl_min),
                    }}
                    onSubmit={handleFormSubmit}
                />
            </div>
        </Card>
    );
};

export default CacheConfig;
