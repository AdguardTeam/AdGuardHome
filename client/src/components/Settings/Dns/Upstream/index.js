import React from 'react';
import PropTypes from 'prop-types';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import Form from './Form';
import Card from '../../../ui/Card';
import { setDnsConfig } from '../../../../actions/dnsConfig';

const Upstream = (props) => {
    const [t] = useTranslation();
    const dispatch = useDispatch();

    const handleSubmit = (values) => {
        dispatch(setDnsConfig(values));
    };

    const {
        processingTestUpstream,
        dnsConfig: {
            upstream_dns,
            bootstrap_dns,
            processingSetConfig,
            upstream_mode,
        },
    } = props;

    return (
        <Card
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
        </Card>
    );
};

Upstream.propTypes = {
    processingTestUpstream: PropTypes.bool.isRequired,
    dnsConfig: PropTypes.object.isRequired,
};

export default Upstream;
