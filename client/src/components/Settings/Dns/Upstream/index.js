import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';
import cn from 'classnames';

import Form from './Form';
import Card from '../../../ui/Card';
import { DNS_REQUEST_OPTIONS } from '../../../../helpers/constants';


class Upstream extends Component {
    handleSubmit = ({ bootstrap_dns, upstream_dns, dnsRequestOption }) => {
        const disabledOption = dnsRequestOption === DNS_REQUEST_OPTIONS.PARALLEL_REQUESTS
            ? DNS_REQUEST_OPTIONS.FASTEST_ADDR
            : DNS_REQUEST_OPTIONS.PARALLEL_REQUESTS;

        const formattedValues = {
            bootstrap_dns,
            upstream_dns,
            [dnsRequestOption]: true,
            [disabledOption]: false,
        };

        this.props.setDnsConfig(formattedValues);
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

        const dnsRequestOption = cn({
            parallel_requests,
            fastest_addr,
        });

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
                                dnsRequestOption,
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

export default withTranslation()(Upstream);
