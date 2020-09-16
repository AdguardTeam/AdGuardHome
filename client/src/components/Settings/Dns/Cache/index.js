import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import { change } from 'redux-form';
import Card from '../../../ui/Card';
import Form from './Form';
import { setDnsConfig } from '../../../../actions/dnsConfig';
import { selectCompletedFields } from '../../../../helpers/helpers';
import { CACHE_CONFIG_FIELDS, FORM_NAME } from '../../../../helpers/constants';

const CacheConfig = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        cache_size, cache_ttl_max, cache_ttl_min,
    } = useSelector((state) => state.dnsConfig, shallowEqual);

    const handleFormSubmit = (values) => {
        const completedFields = selectCompletedFields(values);

        Object.entries(completedFields).forEach(([k, v]) => {
            if ((k === CACHE_CONFIG_FIELDS.cache_ttl_min
                    || k === CACHE_CONFIG_FIELDS.cache_ttl_max)
                    && v === 0) {
                dispatch(change(FORM_NAME.CACHE, k, ''));
            }
        });

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
                        cache_size,
                        cache_ttl_max,
                        cache_ttl_min,
                    }}
                    onSubmit={handleFormSubmit}
                />
            </div>
        </Card>
    );
};

export default CacheConfig;
