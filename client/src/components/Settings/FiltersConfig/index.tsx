import React, { useEffect, useRef } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';

import { toNumber } from '../../../helpers/form';
import { FILTERS_INTERVALS_HOURS, FILTERS_RELATIVE_LINK } from '../../../helpers/constants';
import { Checkbox } from '../../ui/Controls/Checkbox';

const getTitleForInterval = (interval: any, t: any) => {
    if (interval === 0) {
        return t('disabled');
    }
    if (interval === 72 || interval === 168) {
        return t('interval_days', { count: interval / 24 });
    }

    return t('interval_hours', { count: interval });
};

export type FormValues = {
    enabled: boolean;
    interval: number;
};

type Props = {
    initialValues: FormValues;
    setFiltersConfig: (values: FormValues) => void;
    processing: boolean;
};

export const FiltersConfig = ({ initialValues, setFiltersConfig, processing }: Props) => {
    const { t } = useTranslation();
    const prevFormValuesRef = useRef<FormValues>(initialValues);

    const { register, watch, control } = useForm({
        mode: 'onChange',
        defaultValues: initialValues,
    });

    const formValues = watch();

    useEffect(() => {
        const prevFormValues = prevFormValuesRef.current;

        if (JSON.stringify(prevFormValues) !== JSON.stringify(formValues)) {
            setFiltersConfig(formValues);
            prevFormValuesRef.current = formValues;
        }
    }, [formValues]);

    const components = {
        a: <a href={FILTERS_RELATIVE_LINK} rel="noopener noreferrer" />,
    };

    return (
        <>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="enabled"
                            control={control}
                            render={({ field }) => (
                                <Checkbox
                                    {...field}
                                    title={t('block_domain_use_filters_and_hosts')}
                                    disabled={processing}
                                />
                            )}
                        />

                        <p>
                            <Trans components={components}>filters_block_toggle_hint</Trans>
                        </p>
                    </div>
                </div>

                <div className="col-12 col-md-5">
                    <div className="form__group form__group--inner mb-5">
                        <label className="form__label">
                            <Trans>filters_interval</Trans>
                        </label>
                        <select
                            {...register('interval', {
                                setValueAs: toNumber,
                            })}
                            className="custom-select"
                            disabled={processing}>
                            {FILTERS_INTERVALS_HOURS.map((interval) => (
                                <option value={interval} key={interval}>
                                    {getTitleForInterval(interval, t)}
                                </option>
                            ))}
                        </select>
                    </div>
                </div>
            </div>
        </>
    );
};
