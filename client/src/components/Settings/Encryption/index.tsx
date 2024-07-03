import React, { Component } from 'react';
import { withTranslation } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT, ENCRYPTION_SOURCE } from '../../../helpers/constants';

import Form from './Form';

import Card from '../../ui/Card';

import PageTitle from '../../ui/PageTitle';

import Loading from '../../ui/Loading';
import { EncryptionData } from '../../../initialState';

interface EncryptionProps {
    setTlsConfig: (...args: unknown[]) => unknown;
    validateTlsConfig: (...args: unknown[]) => unknown;
    encryption: EncryptionData;
    t: (...args: unknown[]) => string;
}

class Encryption extends Component<EncryptionProps> {
    componentDidMount() {
        const { validateTlsConfig, encryption } = this.props;

        if (encryption.enabled) {
            validateTlsConfig(encryption);
        }
    }

    handleFormSubmit = (values: any) => {
        const submitValues = this.getSubmitValues(values);

        this.props.setTlsConfig(submitValues);
    };

    handleFormChange = debounce((values) => {
        const submitValues = this.getSubmitValues(values);

        if (submitValues.enabled) {
            this.props.validateTlsConfig(submitValues);
        }
    }, DEBOUNCE_TIMEOUT);

    getInitialValues = (data: any) => {
        const { certificate_chain, private_key, private_key_saved } = data;
        const certificate_source = certificate_chain ? ENCRYPTION_SOURCE.CONTENT : ENCRYPTION_SOURCE.PATH;
        const key_source = private_key || private_key_saved ? ENCRYPTION_SOURCE.CONTENT : ENCRYPTION_SOURCE.PATH;

        return {
            ...data,
            certificate_source,
            key_source,
        };
    };

    getSubmitValues = (values: any) => {
        const { certificate_source, key_source, private_key_saved, ...config } = values;

        if (certificate_source === ENCRYPTION_SOURCE.PATH) {
            config.certificate_chain = '';
        } else {
            config.certificate_path = '';
        }

        if (key_source === ENCRYPTION_SOURCE.PATH) {
            config.private_key = '';
        } else {
            config.private_key_path = '';

            if (private_key_saved) {
                config.private_key = '';
                config.private_key_saved = private_key_saved;
            }
        }

        return config;
    };

    render() {
        const { encryption, t } = this.props;
        const {
            enabled,
            server_name,
            force_https,
            port_https,
            port_dns_over_tls,
            port_dns_over_quic,
            certificate_chain,
            private_key,
            certificate_path,
            private_key_path,
            private_key_saved,
            serve_plain_dns,
        } = encryption;

        const initialValues = this.getInitialValues({
            enabled,
            server_name,
            force_https,
            port_https,
            port_dns_over_tls,
            port_dns_over_quic,
            certificate_chain,
            private_key,
            certificate_path,
            private_key_path,
            private_key_saved,
            serve_plain_dns,
        });

        return (
            <div className="encryption">
                <PageTitle title={t('encryption_settings')} />

                {encryption.processing && <Loading />}
                {!encryption.processing && (
                    <Card
                        title={t('encryption_title')}
                        subtitle={t('encryption_desc')}
                        bodyType="card-body box-body--settings">
                        <Form
                            initialValues={initialValues}
                            onSubmit={this.handleFormSubmit}
                            onChange={this.handleFormChange}
                            setTlsConfig={this.props.setTlsConfig}
                            validateTlsConfig={this.props.validateTlsConfig}
                            {...this.props.encryption}
                        />
                    </Card>
                )}
            </div>
        );
    }
}

export default withTranslation()(Encryption);
