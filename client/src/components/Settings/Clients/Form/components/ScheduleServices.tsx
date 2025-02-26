import React from 'react';
import { Trans } from 'react-i18next';
import { useFormContext } from 'react-hook-form';
import { ScheduleForm } from '../../../../Filters/Services/ScheduleForm';
import { ClientForm } from '../types';

export const ScheduleServices = () => {
    const { watch, setValue } = useFormContext<ClientForm>();

    const blockedServicesSchedule = watch('blocked_services_schedule');

    const handleScheduleSubmit = (values: any) => {
        setValue('blocked_services_schedule', values);
    };

    return (
        <>
            <div className="form__desc mb-4">
                <Trans>schedule_services_desc_client</Trans>
            </div>

            <ScheduleForm schedule={blockedServicesSchedule} onScheduleSubmit={handleScheduleSubmit} clientForm />
        </>
    );
};
