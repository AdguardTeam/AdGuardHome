import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';

import Form from './Form';
import Card from '../../ui/Card';

class Services extends Component {
    handleSubmit = (values) => {
        let config = values;

        if (values && values.blocked_services) {
            const blocked_services = Object
                .keys(values.blocked_services)
                .filter(service => values.blocked_services[service]);
            config = blocked_services;
        }

        this.props.setBlockedServices(config);
    };


    getInitialDataForServices = (initial) => {
        if (initial) {
            const blocked = {};

            initial.forEach((service) => {
                blocked[service] = true;
            });

            return {
                blocked_services: blocked,
            };
        }

        return initial;
    };


    render() {
        const { services, t } = this.props;
        const initialData = this.getInitialDataForServices(services.list);

        return (
            <Card
                title={t('blocked_services')}
                subtitle={t('blocked_services_desc')}
                bodyType="card-body box-body--settings"
            >
                <div className="form">
                    <Form
                        initialValues={{ ...initialData }}
                        processing={services.processing}
                        processingSet={services.processingSet}
                        onSubmit={this.handleSubmit}
                    />
                </div>
            </Card>
        );
    }
}

Services.propTypes = {
    t: PropTypes.func.isRequired,
    services: PropTypes.object.isRequired,
    setBlockedServices: PropTypes.func.isRequired,
};

export default withNamespaces()(Services);
