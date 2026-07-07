import { createSignal, createEffect } from 'solid-js';

import intl from 'panel/common/intl';
import { QUERY_LOG_INTERVALS_DAYS, RETENTION_CUSTOM } from 'panel/helpers/constants';

import { RadioGroup } from 'panel/common/ui/SettingsGroup';
import { getIntervalTitle, getDefaultInterval } from '../helpers';
import { RetentionCustomInput } from '../RetentionCustomInput';

export type FormValues = {
    interval: number;
    customInterval?: number | null;
};

type Props = {
    initialValues: Partial<FormValues>;
    processing: boolean;
    onValuesChange: (values: FormValues) => void;
};

export const Form = (props: Props) => {
    const [intervalValue, setIntervalValue] = createSignal(
        getDefaultInterval(props.initialValues.customInterval, props.initialValues.interval),
    );
    const [customInterval, setCustomInterval] = createSignal<number | null>(
        props.initialValues.customInterval ?? null,
    );

    // Clear customInterval when a standard interval is selected
    const handleIntervalChange = (val: number) => {
        const numVal = Number(val);
        setIntervalValue(numVal);
        if (QUERY_LOG_INTERVALS_DAYS.includes(numVal)) {
            setCustomInterval(null);
        }
    };

    // Notify parent of value changes for dirty tracking
    createEffect(() => {
        const values: FormValues = {
            interval: intervalValue(),
            customInterval: customInterval(),
        };
        props.onValuesChange(values);
    });

    return (
        <>
            <RadioGroup
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
        </>
    );
};
