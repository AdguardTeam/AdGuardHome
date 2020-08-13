import React from 'react';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import Form from './Form';
import Card from '../../../ui/Card';
import { setDnsConfig } from '../../../../actions/dnsConfig';

const Upstream = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        upstream_dns,
        bootstrap_dns,
        upstream_mode,
        processingSetConfig,
    } = useSelector((state) => state.dnsConfig, shallowEqual);

    const { processingTestUpstream } = useSelector((state) => state.settings, shallowEqual);

    const handleSubmit = (values) => {
        dispatch(setDnsConfig(values));
    };

    return <Card
        title={t('upstream_dns')}
        subtitle={t('upstream_dns_hint')}
        bodyType="card-body box-body--settings"
    >
        <div className="row">
            <div className="col">
                <Form
                    initialValues={{
                        upstream_dns,
                        bootstrap_dns,
                        upstream_mode,
                    }}
                    onSubmit={handleSubmit}
                    processingTestUpstream={processingTestUpstream}
                    processingSetConfig={processingSetConfig}
                />
            </div>
        </div>
    </Card>;
};

export default Upstream;
