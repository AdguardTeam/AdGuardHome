import { createSignal, createMemo } from 'solid-js';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import theme from 'panel/lib/theme';
import { QUERY_LOG_INTERVALS_DAYS, RETENTION_CUSTOM } from 'panel/helpers/constants';

import { RadioGroup, SwitchGroup } from 'panel/common/ui/SettingsGroup';
import { IgnoredDomains } from '../IgnoredDomains';
import { getIntervalTitle, getDefaultInterval } from '../helpers';
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

export const Form = (props: Props) => {
    const [enabled, setEnabled] = createSignal(props.initialValues.enabled || false);
    const [anonymizeClientIp, setAnonymizeClientIp] = createSignal(
        props.initialValues.anonymize_client_ip || false,
    );
    const [intervalValue, setIntervalValue] = createSignal(
        getDefaultInterval(props.initialValues.customInterval, props.initialValues.interval),
    );
    const [customInterval, setCustomInterval] = createSignal<number | null>(
        props.initialValues.customInterval ?? null,
    );
    const [ignored, setIgnored] = createSignal(props.initialValues.ignored || '');
    const [ignoreEnabled, setIgnoreEnabled] = createSignal(
        props.initialValues.ignore_enabled || true,
    );
    const [isSubmitting, setIsSubmitting] = createSignal(false);

    // Clear customInterval when a standard interval is selected
    const handleIntervalChange = (val: number) => {
        const numVal = Number(val);
        setIntervalValue(numVal);
        if (QUERY_LOG_INTERVALS_DAYS.includes(numVal)) {
            setCustomInterval(null);
        }
    };

    const disableSubmit = createMemo(
        () =>
            isSubmitting() ||
            props.processing ||
            (intervalValue() === RETENTION_CUSTOM && !customInterval()),
    );

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        const data: FormValues = {
            enabled: enabled(),
            anonymize_client_ip: anonymizeClientIp(),
            interval: intervalValue(),
            customInterval: customInterval(),
            ignored: ignored(),
            ignore_enabled: ignoreEnabled(),
        };
        setIsSubmitting(true);
        props.onSubmit(data);
        setIsSubmitting(false);
    };

    return (
        <form onSubmit={handleSubmit}>
            <SwitchGroup
                id="logs_enabled"
                title={intl.getMessage('settings_log_dns_requests')}
                checked={enabled()}
                onChange={(e: Event) => setEnabled((e.target as HTMLInputElement).checked)}
                disabled={props.processing}
            />

            <SwitchGroup
                id="logs_anonymize_client_ip"
                title={intl.getMessage('settings_anonymize_client_ip')}
                description={intl.getMessage('settings_anonymize_client_ip_desc')}
                checked={anonymizeClientIp()}
                onChange={(e: Event) =>
                    setAnonymizeClientIp((e.target as HTMLInputElement).checked)
                }
                disabled={props.processing}
            />

            <RadioGroup
                title={intl.getMessage('query_log_retention')}
                disabled={props.processing}
                value={intervalValue()}
                onChange={handleIntervalChange}
                name="logs-interval"
                options={[
                    { text: getIntervalTitle(RETENTION_CUSTOM), value: RETENTION_CUSTOM },
                    ...QUERY_LOG_INTERVALS_DAYS.map((interval) => ({
                        text: getIntervalTitle(interval),
                        value: interval,
                    })),
                ]}
            >
                <RetentionCustomInput
                    value={customInterval()}
                    onChange={setCustomInterval}
                    processing={props.processing}
                    intervalValue={intervalValue()}
                    intervals={QUERY_LOG_INTERVALS_DAYS}
                    inputId="logs_config_custom_interval"
                    inputLabel={intl.getMessage('settings_log_rotation_hours')}
                    placeholder={intl.getMessage('settings_rotation_placeholder')}
                />
            </RadioGroup>

            <IgnoredDomains
                ignoredValue={ignored()}
                onIgnoredChange={setIgnored}
                processing={props.processing}
                ignoreEnabled={ignoreEnabled()}
                onIgnoreEnabledChange={setIgnoreEnabled}
                switchId="logs_config_ignored_enabled"
                textareaId="logs_config_ignored"
                description={intl.getMessage('ignore_domains_desc_log')}
            />

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="logs_config_save"
                    variant="primary"
                    size="small"
                    disabled={disableSubmit()}
                    class={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    id="logs_config_clear"
                    variant="secondary"
                    size="small"
                    onClick={props.onReset}
                    disabled={props.processingReset}
                    class={theme.form.button}
                >
                    {intl.getMessage('clear_query_log')}
                </Button>
            </div>
        </form>
    );
};
