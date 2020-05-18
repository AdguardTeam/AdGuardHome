import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Form from './Form';
import Card from '../../../ui/Card';

class Upstream extends Component {
    handleSubmit = (values) => {
        this.props.setDnsConfig(values);
    };

    handleTest = (values) => {
        this.props.testUpstream(values);
    };

    render() {
        const {
            t,
            processingTestUpstream,
            dnsConfig: {
                upstream_dns,
                bootstrap_dns,
                fastest_addr,
                parallel_requests,
                processingSetConfig,
            },
        } = this.props;

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
                                fastest_addr,
                                parallel_requests,
                            }}
                            testUpstream={this.handleTest}
                            onSubmit={this.handleSubmit}
                            processingTestUpstream={processingTestUpstream}
                            processingSetConfig={processingSetConfig}
                        />
                    </div>
                </div>
            </Card>
        );
    }
}

Upstream.propTypes = {
    testUpstream: PropTypes.func.isRequired,
    processingTestUpstream: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
    dnsConfig: PropTypes.object.isRequired,
    setDnsConfig: PropTypes.func.isRequired,
};

export default withNamespaces()(Upstream);
