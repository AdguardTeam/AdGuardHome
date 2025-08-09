import React, { useEffect } from 'react';
import i18next from 'i18next';
import { Controller, useForm } from 'react-hook-form';

import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { STATS_INTERVALS_DAYS, DAY, RETENTION_CUSTOM, RETENTION_RANGE } from '../../../helpers/constants';
import { toNumber } from '../../../helpers/form';
import { RadioGroup, SwitchGroup } from '../SettingsGroup';
import { IgnoredDomains } from '../IgnoredDomains';

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
    ignore_enabled: boolean;
};

const defaultFormValues = {
    enabled: false,
    interval: DAY,
    customInterval: null,
    ignored: '',
    ignore_enabled: false,
};

type Props = {
    initialValues: FormValues;
    processing: boolean;
    processingReset: boolean;
    onSubmit: (values: FormValues) => void;
    onReset: () => void;
};

export const Form = ({ initialValues, processing, processingReset, onSubmit, onReset }: Props) => {
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
    const ignoreEnabled = watch('ignore_enabled');

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
            <Controller
                name="enabled"
                control={control}
                render={({ field }) => (
                    <SwitchGroup
                        checked={field.value}
                        onChange={field.onChange}
                        id="stats_config_enabled"
                        title={intl.getMessage('settings_statistics')}
                        description={intl.getMessage('settings_statistics_desc')}
                        disabled={processing}
                    />
                )}
            />

            <RadioGroup
                title={intl.getMessage('settings_statistics_retention')}
                disabled={processing}
                value={intervalValue}
                onChange={(val) => setValue('interval', Number(val))}
                name="stats-interval"
                options={[
                    { text: getIntervalTitle(RETENTION_CUSTOM), value: RETENTION_CUSTOM },
                    ...STATS_INTERVALS_DAYS.map((interval) => ({
                        text: getIntervalTitle(interval),
                        value: interval,
                    })),
                ]}>
                <Controller
                    name="customInterval"
                    control={control}
                    render={({ field, fieldState }) => (
                        <div className={theme.form.input}>
                            <Input
                                id="stats_config_custom_interval"
                                label={intl.getMessage('settings_statistics_retention_hours')}
                                placeholder={intl.getMessage('settings_rotation_placeholder')}
                                value={field.value ?? ''}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                                disabled={processing || STATS_INTERVALS_DAYS.includes(intervalValue)}
                                errorMessage={fieldState.error?.message}
                                min={RETENTION_RANGE.MIN}
                                max={RETENTION_RANGE.MAX}
                            />
                        </div>
                    )}
                />
            </RadioGroup>

            <IgnoredDomains
                control={control}
                processing={processing}
                ignoreEnabled={ignoreEnabled}
                setValue={setValue}
                switchId="stats_config_ignored_enabled"
                textareaId="stats_config_ignored"
            />

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="stats_config_save"
                    variant="primary"
                    size="small"
                    disabled={disableSubmit}
                    className={theme.form.button}>
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    id="stats_config_clear"
                    onClick={onReset}
                    variant="secondary"
                    size="small"
                    disabled={processingReset}
                    className={theme.form.button}>
                    {intl.getMessage('settings_statistics_clear')}
                </Button>
            </div>
        </form>
    );
};
