import React, { useEffect } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import i18next from 'i18next';

import { Controller, useForm } from 'react-hook-form';
import { STATS_INTERVALS_DAYS, DAY, RETENTION_CUSTOM, RETENTION_RANGE } from '../../../helpers/constants';

import '../FormButton.css';
import { Checkbox } from '../../ui/Controls/Checkbox';
import { Input } from '../../ui/Controls/Input';
import { toNumber } from '../../../helpers/form';
import { Textarea } from '../../ui/Controls/Textarea';

const getIntervalTitle = (interval: any) => {
    switch (interval) {
        case RETENTION_CUSTOM:
            return i18next.t('settings_custom');
        case DAY:
            return i18next.t('interval_24_hour');
        default:
            return i18next.t('interval_days', { count: interval / DAY });
    }
};

export type FormValues = {
    enabled: boolean;
    interval: number;
    customInterval?: number | null;
    ignored: string;
};

const defaultFormValues = {
    enabled: false,
    interval: DAY,
    customInterval: null,
    ignored: '',
};

type Props = {
    initialValues: FormValues;
    processing: boolean;
    processingReset: boolean;
    onSubmit: (values: FormValues) => void;
    onReset: () => void;
};

export const Form = ({ initialValues, processing, processingReset, onSubmit, onReset }: Props) => {
    const { t } = useTranslation();

    const {
        handleSubmit,
        watch,
        setValue,
        control,
        formState: { isSubmitting },
    } = useForm<FormValues>({
        mode: 'onBlur',
        defaultValues: {
            ...defaultFormValues,
            ...initialValues,
        },
    });

    const intervalValue = watch('interval');
    const customIntervalValue = watch('customInterval');

    useEffect(() => {
        if (STATS_INTERVALS_DAYS.includes(intervalValue)) {
            setValue('customInterval', null);
        }
    }, [intervalValue]);

    const onSubmitForm = (data: FormValues) => {
        onSubmit(data);
    };

    const disableSubmit = isSubmitting || processing || (intervalValue === RETENTION_CUSTOM && !customIntervalValue);

    return (
        <form onSubmit={handleSubmit(onSubmitForm)}>
            <div className="form__group form__group--settings">
                <Controller
                    name="enabled"
                    control={control}
                    render={({ field }) => (
                        <Checkbox
                            {...field}
                            data-testid="stats_config_enabled"
                            title={t('statistics_enable')}
                            disabled={processing}
                        />
                    )}
                />
            </div>

            <div className="form__label form__label--with-desc">
                <Trans>statistics_retention</Trans>
            </div>

            <div className="form__desc form__desc--top">
                <Trans>statistics_retention_desc</Trans>
            </div>

            <div className="form__group form__group--settings mt-2">
                <div className="custom-controls-stacked">
                    <label className="custom-control custom-radio">
                        <input
                            type="radio"
                            data-testid="stats_config_interval"
                            className="custom-control-input"
                            disabled={processing}
                            checked={!STATS_INTERVALS_DAYS.includes(intervalValue)}
                            value={RETENTION_CUSTOM}
                            onChange={(e) => {
                                setValue('interval', parseInt(e.target.value, 10));
                            }}
                        />

                        <span className="custom-control-label">{getIntervalTitle(RETENTION_CUSTOM)}</span>
                    </label>

                    {!STATS_INTERVALS_DAYS.includes(intervalValue) && (
                        <div className="form__group--input">
                            <div className="form__desc form__desc--top">{i18next.t('custom_retention_input')}</div>

                            <Controller
                                name="customInterval"
                                control={control}
                                render={({ field, fieldState }) => (
                                    <Input
                                        {...field}
                                        data-testid="stats_config_custom_interval"
                                        disabled={processing}
                                        error={fieldState.error?.message}
                                        min={RETENTION_RANGE.MIN}
                                        max={RETENTION_RANGE.MAX}
                                        onChange={(e) => {
                                            const { value } = e.target;
                                            field.onChange(toNumber(value));
                                        }}
                                    />
                                )}
                            />
                        </div>
                    )}
                    {STATS_INTERVALS_DAYS.map((interval) => (
                        <label key={interval} className="custom-control custom-radio">
                            <input
                                type="radio"
                                className="custom-control-input"
                                disabled={processing}
                                value={interval}
                                checked={intervalValue === interval}
                                onChange={(e) => {
                                    setValue('interval', parseInt(e.target.value, 10));
                                }}
                            />

                            <span className="custom-control-label">{getIntervalTitle(interval)}</span>
                        </label>
                    ))}
                </div>
            </div>

            <div className="form__label form__label--with-desc">
                <Trans>ignore_domains_title</Trans>
            </div>

            <div className="form__desc form__desc--top">
                <Trans>ignore_domains_desc_stats</Trans>
            </div>

            <div className="form__group form__group--settings">
                <Controller
                    name="ignored"
                    control={control}
                    render={({ field, fieldState }) => (
                        <Textarea
                            {...field}
                            data-testid="stats_config_ignored"
                            placeholder={t('ignore_domains')}
                            className="text-input"
                            disabled={processing}
                            error={fieldState.error?.message}
                            trimOnBlur
                        />
                    )}
                />
            </div>

            <div className="mt-5">
                <button
                    type="submit"
                    data-testid="stats_config_save"
                    className="btn btn-success btn-standard btn-large"
                    disabled={disableSubmit}>
                    <Trans>save_btn</Trans>
                </button>

                <button
                    type="button"
                    data-testid="stats_config_clear"
                    className="btn btn-outline-secondary btn-standard form__button"
                    onClick={onReset}
                    disabled={processingReset}>
                    <Trans>statistics_clear</Trans>
                </button>
            </div>
        </form>
    );
};
