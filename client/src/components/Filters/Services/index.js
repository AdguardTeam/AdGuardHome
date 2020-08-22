import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { useDispatch, useSelector } from 'react-redux';
import Form from './Form';
import Card from '../../ui/Card';
import { getBlockedServices, setBlockedServices } from '../../../actions/services';
import PageTitle from '../../ui/PageTitle';

const getInitialDataForServices = (initial) => (initial ? initial.reduce(
    (acc, service) => {
        acc.blocked_services[service] = true;
        return acc;
    }, { blocked_services: {} },
) : initial);

const Services = () => {
    const [t] = useTranslation();
    const dispatch = useDispatch();
    const services = useSelector((store) => store?.services);

    useEffect(() => {
        dispatch(getBlockedServices());
    }, []);

    const handleSubmit = (values) => {
        if (!values || !values.blocked_services) {
            return;
        }

        const blocked_services = Object
            .keys(values.blocked_services)
            .filter((service) => values.blocked_services[service]);

        dispatch(setBlockedServices(blocked_services));
    };

    const initialValues = getInitialDataForServices(services.list);

    return (
        <>
            <PageTitle
                title={t('blocked_services')}
                subtitle={t('blocked_services_desc')}
            />
            <Card
                bodyType="card-body box-body--settings"
            >
                <div className="form">
                    <Form
                        initialValues={initialValues}
                        processing={services.processing}
                        processingSet={services.processingSet}
                        onSubmit={handleSubmit}
                    />
                </div>
            </Card>
        </>
    );
};

export default Services;
