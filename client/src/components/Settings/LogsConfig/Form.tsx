import React, { useEffect } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import i18next from 'i18next';
import { Controller, useForm } from 'react-hook-form';

import { trimLinesAndRemoveEmpty } from '../../../helpers/helpers';
import {
    QUERY_LOG_INTERVALS_DAYS,
    HOUR,
    DAY,
    RETENTION_CUSTOM,
    RETENTION_RANGE,
    CUSTOM_INTERVAL,
} from '../../../helpers/constants';
import '../FormButton.css';
import { Checkbox } from '../../ui/Controls/Checkbox';

const getIntervalTitle = (interval: number) => {
    switch (interval) {
        case RETENTION_CUSTOM:
            return i18next.t('settings_custom');
        case 6 * HOUR:
            return i18next.t('interval_6_hour');
        case DAY:
            return i18next.t('interval_24_hour');
        default:
            return i18next.t('interval_days', { count: interval / DAY });
    }
};

export type FormValues = {
    enabled: boolean;
    anonymize_client_ip: boolean;
    interval: number;
    customInterval?: number | null;
    ignored: string;
};

type Props = {
    initialValues: Partial<FormValues>;
    processing: boolean;
    processingReset: boolean;
    onSubmit: (values: FormValues) => void;
    onReset: () => void;
};

export const Form = ({ initialValues, processing, processingReset, onSubmit, onReset }: Props) => {
    const { t } = useTranslation();

    const {
        register,
        handleSubmit,
        watch,
        setValue,
        control,
        formState: { isSubmitting },
    } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: {
            enabled: initialValues.enabled || false,
            anonymize_client_ip: initialValues.anonymize_client_ip || false,
            interval: initialValues.interval || DAY,
            customInterval: initialValues.customInterval || null,
            ignored: initialValues.ignored || '',
        },
    });

    const intervalValue = watch('interval');
    const customIntervalValue = watch('customInterval');

    useEffect(() => {
        if (QUERY_LOG_INTERVALS_DAYS.includes(intervalValue)) {
            setValue('customInterval', null);
        }
    }, [intervalValue]);

    const onSubmitForm = (data: FormValues) => {
        onSubmit(data);
    };

    const handleIgnoredBlur = (e: React.FocusEvent<HTMLTextAreaElement>) => {
        const trimmed = trimLinesAndRemoveEmpty(e.target.value);
        setValue('ignored', trimmed);
    };

    const disableSubmit = isSubmitting || processing || (intervalValue === RETENTION_CUSTOM && !customIntervalValue);

    return (
        <form onSubmit={handleSubmit(onSubmitForm)}>
            <div className="form__group form__group--settings">
                <Controller
                    name="enabled"
                    control={control}
                    render={({ field }) => <Checkbox {...field} title={t('query_log_enable')} disabled={processing} />}
                />
            </div>

            <div className="form__group form__group--settings">
                <Controller
                    name="anonymize_client_ip"
                    control={control}
                    render={({ field }) => (
                        <Checkbox
                            {...field}
                            title={t('anonymize_client_ip')}
                            subtitle={t('anonymize_client_ip_desc')}
                            disabled={processing}
                        />
                    )}
                />
            </div>

            <div className="form__label">
                <Trans>query_log_retention</Trans>
            </div>

            <div className="form__group form__group--settings">
                <div className="custom-controls-stacked">
                    <label className="custom-control custom-radio">
                        <input
                            type="radio"
                            className="custom-control-input"
                            disabled={processing}
                            checked={!QUERY_LOG_INTERVALS_DAYS.includes(intervalValue)}
                            value={RETENTION_CUSTOM}
                            onChange={(e) => {
                                setValue('interval', parseInt(e.target.value, 10));
                            }}
                        />

                        <span className="custom-control-label">{getIntervalTitle(RETENTION_CUSTOM)}</span>
                    </label>

                    {!QUERY_LOG_INTERVALS_DAYS.includes(intervalValue) && (
                        <div className="form__group--input">
                            <div className="form__desc form__desc--top">{i18next.t('custom_rotation_input')}</div>

                            <input
                                type="number"
                                className="form-control"
                                name={CUSTOM_INTERVAL}
                                min={RETENTION_RANGE.MIN}
                                max={RETENTION_RANGE.MAX}
                                disabled={processing}
                                {...register('customInterval')}
                                onChange={(e) => {
                                    setValue('customInterval', parseInt(e.target.value, 10));
                                }}
                            />
                        </div>
                    )}

                    {QUERY_LOG_INTERVALS_DAYS.map((interval) => (
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

            <label className="form__label form__label--with-desc">
                <Trans>ignore_domains_title</Trans>
            </label>

            <div className="form__desc form__desc--top">
                <Trans>ignore_domains_desc_query</Trans>
            </div>

            <div className="form__group form__group--settings">
                <textarea
                    className="form-control form-control--textarea font-monospace text-input"
                    placeholder={i18next.t('ignore_domains')}
                    disabled={processing}
                    {...register('ignored')}
                    onBlur={handleIgnoredBlur}
                />
            </div>

            <div className="mt-5">
                <button type="submit" className="btn btn-success btn-standard btn-large" disabled={disableSubmit}>
                    <Trans>save_btn</Trans>
                </button>

                <button
                    type="button"
                    className="btn btn-outline-secondary btn-standard form__button"
                    onClick={onReset}
                    disabled={processingReset}>
                    <Trans>query_log_clear</Trans>
                </button>
            </div>
        </form>
    );
};
