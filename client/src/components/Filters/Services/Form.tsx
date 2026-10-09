import React, { useMemo, useState } from 'react';

import { Trans, useTranslation } from 'react-i18next';

import { Controller, useForm } from 'react-hook-form';

import { ServiceField } from './ServiceField';
import { ScheduleForm } from './ScheduleForm';

export type BlockedService = {
    id: string;
    name: string;
    icon_svg: string;
    group_id: string;
};

export type ServiceGroups = {
    id: string;
}

export type ServiceSchedule = {
    id: string;
    schedule?: {
        time_zone: string;
        [key: string]: any;
    };
};

type FormValues = {
    blocked_services: Record<string, boolean>;
};

interface FormProps {
    initialValues: Record<string, boolean>;
    blockedServices: BlockedService[];
    serviceGroups: ServiceGroups[];
    serviceSchedules?: ServiceSchedule[];
    onSubmit: (values: FormValues & { services?: ServiceSchedule[] }) => void;
    onScheduleSubmit?: (serviceId: string, schedule: any) => void;
    processing: boolean;
    processingSet: boolean;
}

export const Form = ({
    initialValues,
    blockedServices,
    serviceGroups,
    serviceSchedules = [],
    onSubmit,
    onScheduleSubmit,
    processing,
    processingSet,
}: FormProps) => {
    const { t } = useTranslation();
    const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
    const [currentServiceId, setCurrentServiceId] = useState<string | null>(null);

    const {
        handleSubmit,
        control,
        setValue,
        formState: { isSubmitting },
    } = useForm<FormValues>({
        mode: 'onBlur',
        defaultValues: { blocked_services: initialValues }
    });

    const isServicesControlsDisabled = processing || processingSet;
    const isSubmitDisabled = processing || processingSet || isSubmitting;

    const servicesByGroup = useMemo(() => {
        return blockedServices.reduce((acc, service) => {
            if (!acc[service.group_id]) {
                acc[service.group_id] = [];
            }
            acc[service.group_id].push(service);
            return acc;
        }, {} as Record<string, BlockedService[]>);
    }, [blockedServices]);

    const serviceScheduleMap = useMemo(() => {
        const map: Record<string, any> = {};
        serviceSchedules.forEach(svc => {
            map[svc.id] = svc.schedule;
        });
        return map;
    }, [serviceSchedules]);

    const handleToggleAllServices = (isSelected: boolean) => {
        blockedServices.forEach((service) => {
            if (!isServicesControlsDisabled) {
                setValue(`blocked_services.${service.id}`, isSelected);
            }
        });
    };

    const handleToggleGroupServices = (groupId: string, isSelected: boolean) => {
        if (isServicesControlsDisabled) {
            return;
        }
        servicesByGroup[groupId].forEach((service) => {
            setValue(`blocked_services.${service.id}`, isSelected);
        });
    };

    const handleScheduleClick = (serviceId: string) => {
        setCurrentServiceId(serviceId);
        setScheduleModalOpen(true);
    };

    const handleScheduleSubmit = (schedule: any) => {
        if (currentServiceId && onScheduleSubmit) {
            onScheduleSubmit(currentServiceId, schedule);
        }
        setScheduleModalOpen(false);
        setCurrentServiceId(null);
    };

    const handleDeleteSchedule = (serviceId: string) => {
        if (onScheduleSubmit) {
            onScheduleSubmit(serviceId, null);
        }
        setScheduleModalOpen(false);
        setCurrentServiceId(null);
    };

    const handleSubmitWithGroups = (values: FormValues) => {
        if (!values || !values.blocked_services) {
            return onSubmit(values);
        }

        const enabledIdsMap = Object.fromEntries(
            blockedServices
                .filter(service => values.blocked_services?.[service.id])
                .map(service => [service.id, true] as const)
        );

        return onSubmit({ blocked_services: enabledIdsMap });
    };

    const currentServiceSchedule = currentServiceId ? serviceScheduleMap[currentServiceId] : null;

    return (
        <form onSubmit={handleSubmit(handleSubmitWithGroups)}>
            <div className="form__group">
                <div className="blocked_services row mb-5">
                    <div className="col-12 col-md-6 mb-4 mb-md-0">
                        <button
                            type="button"
                            data-testid="blocked_services_block_all"
                            className="btn btn-secondary btn-block font-weight-normal"
                            disabled={isServicesControlsDisabled}
                            onClick={() => handleToggleAllServices(true)}>
                            <Trans>block_all</Trans>
                        </button>
                    </div>
                    <div className="col-12 col-md-6">
                        <button
                            type="button"
                            data-testid="blocked_services_unblock_all"
                            className="btn btn-secondary btn-block font-weight-normal"
                            disabled={isServicesControlsDisabled}
                            onClick={() => handleToggleAllServices(false)}>
                            <Trans>unblock_all</Trans>
                        </button>
                    </div>
                </div>

                {serviceGroups.map((group) => {
                    const groupServices = servicesByGroup[group.id];

                    return (
                        <div key={group.id} className="services-group mb-2">
                            <h3 className="h5 mb-3">
                                {t(`servicesgroup.${group.id}.name`, { ns: 'services' })}
                            </h3>

                            {groupServices.length > 1 && (
                                <div className="actions mb-3 d-flex gap-4">
                                    <button
                                        type="button"
                                        className="btn btn-link p-0 text-danger font-weight-normal mr-5"
                                        disabled={isServicesControlsDisabled}
                                        onClick={() => handleToggleGroupServices(group.id, true)}
                                    >
                                        <Trans>block_all</Trans>
                                    </button>

                                    <button
                                        type="button"
                                        className="btn btn-link p-0 text-success font-weight-normal"
                                        disabled={isServicesControlsDisabled}
                                        onClick={() => handleToggleGroupServices(group.id, false)}
                                    >
                                        <Trans>unblock_all</Trans>
                                    </button>
                                </div>
                            )}

                            <div className="services__wrapper">
                                <div className="services">
                                    {groupServices.map((service) => (
                                        <Controller
                                            key={service.id}
                                            name={`blocked_services.${service.id}`}
                                            control={control}
                                            render={({ field }) => (
                                                <ServiceField
                                                    {...field}
                                                    data-testid={`blocked_services_${service.id}`}
                                                    data-groupid={`blocked_services_${service.group_id}`}
                                                    placeholder={service.name}
                                                    disabled={isServicesControlsDisabled}
                                                    icon={service.icon_svg}
                                                    hasSchedule={!!serviceScheduleMap[service.id]}
                                                    onScheduleClick={() => handleScheduleClick(service.id)}
                                                />
                                            )}
                                        />
                                    ))}
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>

            <div className="btn-list">
                <button
                    type="submit"
                    data-testid="blocked_services_save"
                    className="btn btn-success btn-standard btn-large"
                    disabled={isSubmitDisabled}>
                    <Trans>save_btn</Trans>
                </button>
            </div>

            {scheduleModalOpen && (
                <div className="modal d-block" style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>
                    <div className="modal-dialog modal-lg">
                        <div className="modal-content">
                            <div className="modal-header">
                                <h5 className="modal-title">
                                    {t('schedule_services')} - {currentServiceId}
                                </h5>
                                <button
                                    type="button"
                                    className="close"
                                    onClick={() => setScheduleModalOpen(false)}
                                >
                                    <span>&times;</span>
                                </button>
                            </div>
                            <div className="modal-body">
                                <ScheduleForm
                                    schedule={currentServiceSchedule || { time_zone: 'Local' }}
                                    onScheduleSubmit={handleScheduleSubmit}
                                />
                                {currentServiceSchedule && (
                                    <button
                                        type="button"
                                        className="btn btn-danger mt-3"
                                        onClick={() => handleDeleteSchedule(currentServiceId!)}
                                    >
                                        <Trans>schedule_remove</Trans>
                                    </button>
                                )}
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </form>
    );
};
