import React from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Card from '../../../ui/Card';
import Form from './Form';

const Config = ({ t, dnsConfig, setDnsConfig }) => {
    const handleFormSubmit = (values) => {
        setDnsConfig(values);
    };

    const {
        blocking_mode,
        ratelimit,
        blocking_ipv4,
        blocking_ipv6,
        edns_cs_enabled,
        dnssec_enabled,
        disable_ipv6,
        processingSetConfig,
    } = dnsConfig;

    return (
        <Card
            title={t('dns_config')}
            bodyType="card-body box-body--settings"
            id="dns-config"
        >
            <div className="form">
                <Form
                    initialValues={{
                        ratelimit,
                        blocking_mode,
                        blocking_ipv4,
                        blocking_ipv6,
                        edns_cs_enabled,
                        disable_ipv6,
                        dnssec_enabled,
                    }}
                    onSubmit={handleFormSubmit}
                    processing={processingSetConfig}
                />
            </div>
        </Card>
    );
};

Config.propTypes = {
    dnsConfig: PropTypes.object.isRequired,
    setDnsConfig: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Config);
