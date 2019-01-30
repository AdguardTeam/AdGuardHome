import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Form from './Form';
import Card from '../../ui/Card';

class Encryption extends Component {
    handleFormSubmit = (values) => {
        this.props.setTlsConfig(values);
    };

    render() {
        const { encryption, t } = this.props;
        const {
            processing,
            processingConfig,
            status_cert: statusCert,
            status_key: statusKey,
            ...values
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
                            initialValues={{ ...values }}
                            processing={encryption.processingConfig}
                            statusCert={statusCert}
                            statusKey={statusKey}
                            onSubmit={this.handleFormSubmit}
                        />
                    </Card>
                }
            </div>
        );
    }
}

Encryption.propTypes = {
    setTlsConfig: PropTypes.func.isRequired,
    encryption: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Encryption);
