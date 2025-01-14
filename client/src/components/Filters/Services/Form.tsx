import React from 'react';

import { Trans } from 'react-i18next';

import { Controller, useForm } from 'react-hook-form';

import { ServiceField } from './ServiceField';

type BlockedService = {
    id: string;
    name: string;
    icon_svg: string;
}

type FormValues = {
    blocked_services: Record<string, boolean>;
}

interface FormProps {
    initialValues: Record<string, boolean>;
    blockedServices: BlockedService[];
    onSubmit: (...args: unknown[]) => void;
    processing: boolean;
    processingSet: boolean;
}

export const Form = ({ initialValues, blockedServices, processing, processingSet, onSubmit }: FormProps) => {
    const { handleSubmit, control, setValue, formState: { isSubmitting, isDirty } } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: initialValues,
    });

    const handleToggleAllServices = (isSelected: boolean) => {
        blockedServices.forEach((service: BlockedService) => setValue(`blocked_services.${service.id}`, isSelected));
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="form__group">
                <div className="row mb-4">
                    <div className="col-6">
                        <button
                            type="button"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => handleToggleAllServices(true)}
                        >
                            <Trans>block_all</Trans>
                        </button>
                    </div>

                    <div className="col-6">
                        <button
                            type="button"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => handleToggleAllServices(false)}
                        >
                            <Trans>unblock_all</Trans>
                        </button>
                    </div>
                </div>

                <div className="services">
                    {blockedServices.map((service: any) => (
                        <Controller
                            key={service.id}
                            name={`blocked_services.${service.id}`}
                            control={control}
                            render={({ field }) => (
                                <ServiceField
                                    {...field}
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
                    className="btn btn-success btn-standard btn-large"
                    disabled={isSubmitting || !isDirty || processing || processingSet}>
                    <Trans>save_btn</Trans>
                </button>
            </div>
        </form>
    );
};
