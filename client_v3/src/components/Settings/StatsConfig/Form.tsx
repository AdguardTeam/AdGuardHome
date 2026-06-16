import { createSignal, createMemo } from 'solid-js';

import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { RadioGroup, SwitchGroup } from 'panel/common/ui/SettingsGroup';

import { getIntervalTitle, getDefaultInterval } from '../helpers';
import { STATS_INTERVALS_DAYS, RETENTION_CUSTOM } from '../../../helpers/constants';
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

export const Form = (props: Props) => {
    const [enabled, setEnabled] = createSignal(props.initialValues.enabled || false);
    const [intervalValue, setIntervalValue] = createSignal(
        getDefaultInterval(props.initialValues.customInterval, props.initialValues.interval),
    );
    const [customInterval, setCustomInterval] = createSignal<number | null>(
        props.initialValues.customInterval ?? null,
    );
    const [ignored, setIgnored] = createSignal(props.initialValues.ignored || '');
    const [ignoreEnabled, setIgnoreEnabled] = createSignal(props.initialValues.ignore_enabled || true);
    const [isSubmitting, setIsSubmitting] = createSignal(false);

    // Clear customInterval when a standard interval is selected
    const handleIntervalChange = (val: number) => {
        const numVal = Number(val);
        setIntervalValue(numVal);
        if (STATS_INTERVALS_DAYS.includes(numVal)) {
            setCustomInterval(null);
        }
    };

    const disableSubmit = createMemo(
        () => isSubmitting() || props.processing || (intervalValue() === RETENTION_CUSTOM && !customInterval()),
    );

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        const data: FormValues = {
            enabled: enabled(),
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
                checked={enabled()}
                onChange={(e: Event) => setEnabled((e.target as HTMLInputElement).checked)}
                id="stats_config_enabled"
                title={intl.getMessage('settings_statistics')}
                description={intl.getMessage('settings_statistics_desc')}
                disabled={props.processing}
            />

            <RadioGroup
                title={intl.getMessage('settings_statistics_retention')}
                disabled={props.processing}
                value={intervalValue()}
                onChange={handleIntervalChange}
                name="stats-interval"
                options={[
                    { text: getIntervalTitle(RETENTION_CUSTOM), value: RETENTION_CUSTOM },
                    ...STATS_INTERVALS_DAYS.map((interval) => ({
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
                    intervals={STATS_INTERVALS_DAYS}
                    inputId="stats_config_custom_interval"
                    inputLabel={intl.getMessage('settings_statistics_retention_hours')}
                    placeholder={intl.getMessage('settings_rotation_placeholder')}
                />
            </RadioGroup>

            <IgnoredDomains
                ignoredValue={ignored()}
                onIgnoredChange={setIgnored}
                processing={props.processing}
                ignoreEnabled={ignoreEnabled()}
                onIgnoreEnabledChange={setIgnoreEnabled}
                switchId="stats_config_ignored_enabled"
                textareaId="stats_config_ignored"
                description={intl.getMessage('ignore_domains_desc_stats')}
            />

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="stats_config_save"
                    variant="primary"
                    size="small"
                    disabled={disableSubmit()}
                    class={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    id="stats_config_clear"
                    onClick={props.onReset}
                    variant="secondary"
                    size="small"
                    disabled={props.processingReset}
                    class={theme.form.button}
                >
                    {intl.getMessage('settings_statistics_clear')}
                </Button>
            </div>
        </form>
    );
};
