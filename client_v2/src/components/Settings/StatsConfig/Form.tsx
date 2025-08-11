import React, { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';

import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { getIntervalTitle, getDefaultInterval } from '../helpers';

import { STATS_INTERVALS_DAYS, RETENTION_CUSTOM } from '../../../helpers/constants';
import { RadioGroup, SwitchGroup } from '../SettingsGroup';
import { IgnoredDomains } from '../IgnoredDomains';
import { RetentionCustomInput } from '../RetentionCustomInput';

export type FormValues = {
    enabled: boolean;
    interval: number;
    customInterval?: number | null;
    ignored: string;
    ignore_enabled: boolean;
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
            enabled: initialValues.enabled || false,
            interval: getDefaultInterval(initialValues.customInterval, initialValues.interval),
            customInterval: initialValues.customInterval ?? undefined,
            ignored: initialValues.ignored || '',
            ignore_enabled: initialValues.ignore_enabled || true,
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

    // Focus is handled inside RetentionCustomInput

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
                <RetentionCustomInput
                    control={control}
                    processing={processing}
                    intervalValue={intervalValue}
                    intervals={STATS_INTERVALS_DAYS}
                    inputId="stats_config_custom_interval"
                    inputLabel={intl.getMessage('settings_statistics_retention_hours')}
                    placeholder={intl.getMessage('settings_rotation_placeholder')}
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
