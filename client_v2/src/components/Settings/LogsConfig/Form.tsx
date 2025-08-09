import React, { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import theme from 'panel/lib/theme';
import { QUERY_LOG_INTERVALS_DAYS, DAY, RETENTION_CUSTOM } from 'panel/helpers/constants';

import { RadioGroup } from '../SettingsGroup/RadioGroup';
import { SwitchGroup } from '../SettingsGroup';
import { IgnoredDomains } from '../IgnoredDomains';
import { getIntervalTitle } from '../helpers';
import { RetentionCustomInput } from '../RetentionCustomInput';

export type FormValues = {
    enabled: boolean;
    anonymize_client_ip: boolean;
    interval: number;
    customInterval?: number | null;
    ignored: string;
    ignore_enabled: boolean;
};

type Props = {
    initialValues: Partial<FormValues>;
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
            anonymize_client_ip: initialValues.anonymize_client_ip || false,
            interval: initialValues.interval || DAY,
            customInterval: initialValues.customInterval ?? undefined,
            ignored: initialValues.ignored || '',
            ignore_enabled: initialValues.ignore_enabled || true,
        },
    });

    const intervalValue = watch('interval');
    const customIntervalValue = watch('customInterval');
    const ignoreEnabled = watch('ignore_enabled');

    useEffect(() => {
        if (QUERY_LOG_INTERVALS_DAYS.includes(intervalValue)) {
            setValue('customInterval', undefined);
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
                        id="logs_enabled"
                        title={intl.getMessage('settings_log_dns_requests')}
                        checked={field.value}
                        onChange={field.onChange}
                        disabled={processing}
                    />
                )}
            />

            <Controller
                name="anonymize_client_ip"
                control={control}
                render={({ field }) => (
                    <SwitchGroup
                        id="logs_anonymize_client_ip"
                        title={intl.getMessage('settings_anonymize_client_ip')}
                        description={intl.getMessage('settings_anonymize_client_ip_desc')}
                        checked={field.value}
                        onChange={field.onChange}
                        disabled={processing}
                    />
                )}
            />

            <RadioGroup
                title={intl.getMessage('query_log_retention')}
                disabled={processing}
                value={intervalValue}
                onChange={(val) => setValue('interval', Number(val))}
                name="logs-interval"
                options={[
                    { text: getIntervalTitle(RETENTION_CUSTOM), value: RETENTION_CUSTOM },
                    ...QUERY_LOG_INTERVALS_DAYS.map((interval) => ({
                        text: getIntervalTitle(interval),
                        value: interval,
                    })),
                ]}>
                <RetentionCustomInput
                    control={control}
                    processing={processing}
                    intervalValue={intervalValue}
                    intervals={QUERY_LOG_INTERVALS_DAYS}
                    inputId="logs_config_custom_interval"
                    inputLabel={intl.getMessage('settings_log_rotation_hours')}
                    placeholder={intl.getMessage('settings_rotation_placeholder')}
                />
            </RadioGroup>

            <IgnoredDomains
                control={control}
                processing={processing}
                ignoreEnabled={ignoreEnabled}
                setValue={setValue}
                switchId="logs_config_ignored_enabled"
                textareaId="logs_config_ignored"
            />

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    data-testid="logs_config_save"
                    variant="primary"
                    size="small"
                    disabled={disableSubmit}
                    className={theme.form.button}>
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    data-testid="logs_config_clear"
                    variant="secondary"
                    size="small"
                    onClick={onReset}
                    disabled={processingReset}
                    className={theme.form.button}>
                    {intl.getMessage('clear_query_log')}
                </Button>
            </div>
        </form>
    );
};
