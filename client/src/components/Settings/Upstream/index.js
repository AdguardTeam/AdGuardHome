import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Form from './Form';
import Card from '../../ui/Card';

class Upstream extends Component {
    handleSubmit = (values) => {
        this.props.setUpstream(values);
    };

    handleTest = (values) => {
        this.props.testUpstream(values);
    }

    render() {
        const {
            t,
            upstreamDns: upstream_dns,
            bootstrapDns: bootstrap_dns,
            allServers: all_servers,
            processingSetUpstream,
            processingTestUpstream,
        } = this.props;

        return (
            <Card
                title={ t('upstream_dns') }
                subtitle={ t('upstream_dns_hint') }
                bodyType="card-body box-body--settings"
            >
                <div className="row">
                    <div className="col">
                        <Form
                            initialValues={{
                                upstream_dns,
                                bootstrap_dns,
                                all_servers,
                            }}
                            testUpstream={this.handleTest}
                            onSubmit={this.handleSubmit}
                            processingTestUpstream={processingTestUpstream}
                            processingSetUpstream={processingSetUpstream}
                        />
                    </div>
                </div>
            </Card>
        );
    }
}

Upstream.propTypes = {
    upstreamDns: PropTypes.string,
    bootstrapDns: PropTypes.string,
    allServers: PropTypes.bool,
    setUpstream: PropTypes.func.isRequired,
    testUpstream: PropTypes.func.isRequired,
    processingSetUpstream: PropTypes.bool.isRequired,
    processingTestUpstream: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Upstream);
