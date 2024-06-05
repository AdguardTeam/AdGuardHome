import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { useDispatch, useSelector } from 'react-redux';

import Form from './Form';

import Card from '../../ui/Card';
import { getBlockedServices, getAllBlockedServices, updateBlockedServices } from '../../../actions/services';

import PageTitle from '../../ui/PageTitle';

import { ScheduleForm } from './ScheduleForm';
import { RootState } from '../../../initialState';

const getInitialDataForServices = (initial: any) =>
    initial
        ? initial.reduce(
              (acc: any, service: any) => {
                  acc.blocked_services[service] = true;
                  return acc;
              },
              { blocked_services: {} },
          )
        : initial;

const Services = () => {
    const [t] = useTranslation();
    const dispatch = useDispatch();

    const services = useSelector((state: RootState) => state.services);

    useEffect(() => {
        dispatch(getBlockedServices());
        dispatch(getAllBlockedServices());
    }, []);

    const handleSubmit = (values: any) => {
        if (!values || !values.blocked_services) {
            return;
        }

        const blocked_services = Object.keys(values.blocked_services).filter(
            (service) => values.blocked_services[service],
        );

        dispatch(
            updateBlockedServices({
                ids: blocked_services,
                schedule: services.list.schedule,
            }),
        );
    };

    const handleScheduleSubmit = (values: any) => {
        dispatch(
            updateBlockedServices({
                ids: services.list.ids,
                schedule: values,
            }),
        );
    };

    const initialValues = getInitialDataForServices(services.list.ids);

    if (!initialValues) {
        return null;
    }

    return (
        <>
            <PageTitle title={t('blocked_services')} subtitle={t('blocked_services_desc')} />

            <Card bodyType="card-body box-body--settings">
                <div className="form">
                    <Form
                        initialValues={initialValues}
                        blockedServices={services.allServices}
                        processing={services.processing}
                        processingSet={services.processingSet}
                        onSubmit={handleSubmit}
                    />
                </div>
            </Card>

            <Card
                title={t('schedule_services')}
                subtitle={t('schedule_services_desc')}
                bodyType="card-body box-body--settings">
                <ScheduleForm schedule={services.list.schedule} onScheduleSubmit={handleScheduleSubmit} />
            </Card>
        </>
    );
};

export default Services;
