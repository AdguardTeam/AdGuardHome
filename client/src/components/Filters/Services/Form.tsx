import React from 'react';

import { Trans } from 'react-i18next';

import { Controller, useForm } from 'react-hook-form';

import { ServiceField } from './ServiceField';

export type BlockedService = {
    id: string;
    name: string;
    icon_svg: string;
};

type FormValues = {
    blocked_services: Record<string, boolean>;
};

interface FormProps {
    initialValues: Record<string, boolean>;
    blockedServices: BlockedService[];
    onSubmit: (values: FormValues) => void;
    processing: boolean;
    processingSet: boolean;
}

export const Form = ({ initialValues, blockedServices, processing, processingSet, onSubmit }: FormProps) => {
    const {
        handleSubmit,
        control,
        setValue,
        formState: { isSubmitting },
    } = useForm<FormValues>({
        mode: 'onBlur',
        defaultValues: initialValues,
    });

    const handleToggleAllServices = async (isSelected: boolean) => {
        blockedServices.forEach((service: BlockedService) => setValue(`blocked_services.${service.id}`, isSelected));
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="form__group">
                <div className="row mb-4">
                    <div className="col-6">
                        <button
                            type="button"
                            data-testid="blocked_services_block_all"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => handleToggleAllServices(true)}>
                            <Trans>block_all</Trans>
                        </button>
                    </div>

                    <div className="col-6">
                        <button
                            type="button"
                            data-testid="blocked_services_unblock_all"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => handleToggleAllServices(false)}>
                            <Trans>unblock_all</Trans>
                        </button>
                    </div>
                </div>

                <div className="services">
                    {blockedServices.map((service: BlockedService) => (
                        <Controller
                            key={service.id}
                            name={`blocked_services.${service.id}`}
                            control={control}
                            render={({ field }) => (
                                <ServiceField
                                    {...field}
                                    data-testid={`blocked_services_${service.id}`}
                                    placeholder={service.name}
                                    disabled={processing || processingSet}
                                    icon={service.icon_svg}
                                />
                            )}
                        />
                    ))}
                </div>
            </div>

            <div className="btn-list">
                <button
                    type="submit"
                    data-testid="blocked_services_save"
                    className="btn btn-success btn-standard btn-large"
                    disabled={isSubmitting || processing || processingSet}>
                    <Trans>save_btn</Trans>
                </button>
            </div>
        </form>
    );
};
