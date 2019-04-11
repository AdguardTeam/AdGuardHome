import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';
import debounce from 'lodash/debounce';

import { DEBOUNCE_TIMEOUT } from '../../../helpers/constants';
import Form from './Form';
import Card from '../../ui/Card';

class Encryption extends Component {
    componentDidMount() {
        if (this.props.encryption.enabled) {
            this.props.validateTlsConfig(this.props.encryption);
        }
    }

    handleFormSubmit = (values) => {
        this.props.setTlsConfig(values);
    };

    handleFormChange = debounce((values) => {
        this.props.validateTlsConfig(values);
    }, DEBOUNCE_TIMEOUT);

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
                            onSubmit={this.handleFormSubmit}
                            onChange={this.handleFormChange}
                            setTlsConfig={this.props.setTlsConfig}
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
