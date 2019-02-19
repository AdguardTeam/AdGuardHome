import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';
import debounce from 'lodash/debounce';

import Form from './Form';
import Card from '../../ui/Card';

class Encryption extends Component {
    handleFormSubmit = (values) => {
        this.props.setTlsConfig(values);
    };

    handleFormChange = debounce((values) => {
        this.props.validateTlsConfig(values);
    }, 300);

    render() {
        const { encryption, t } = this.props;
        const {
            enabled,
            server_name,
            force_https,
            port_https,
            port_dns_over_tls,
            certificate_chain,
            private_key,
        } = encryption;

        return (
            <div className="encryption">
                {encryption &&
                    <Card
                        title={t('encryption_title')}
                        subtitle={t('encryption_desc')}
                        bodyType="card-body box-body--settings"
                    >
                        <Form
                            initialValues={{
                                enabled,
                                server_name,
                                force_https,
                                port_https,
                                port_dns_over_tls,
                                certificate_chain,
                                private_key,
                            }}
                            processing={encryption.processingConfig}
                            processingValidate={encryption.processingValidate}
                            onSubmit={this.handleFormSubmit}
                            onChange={this.handleFormChange}
                            {...this.props.encryption}
                        />
                    </Card>
                }
            </div>
        );
    }
}

Encryption.propTypes = {
    setTlsConfig: PropTypes.func.isRequired,
    validateTlsConfig: PropTypes.func.isRequired,
    encryption: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Encryption);
