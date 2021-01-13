import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import Form from './Form';
import Card from '../../../ui/Card';
import { setDnsConfig } from '../../../../actions/dnsConfig';

const RebindingConfig = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        rebinding_protection_enabled, rebinding_allowed_hosts,
    } = useSelector((state) => state.dnsConfig, shallowEqual);

    const handleFormSubmit = (values) => {
        dispatch(setDnsConfig(values));
    };

    return (
        <Card
            title={t('rebinding_title')}
            subtitle={t('rebinding_desc')}
            bodyType="card-body box-body--settings"
        >
            <Form
                initialValues={{
                    rebinding_protection_enabled,
                    rebinding_allowed_hosts,
                }}
                onSubmit={handleFormSubmit}
            />
        </Card>
    );
};

export default RebindingConfig;
