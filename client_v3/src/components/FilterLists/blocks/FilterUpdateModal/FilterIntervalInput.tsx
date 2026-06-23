import { createEffect, createSignal, untrack } from 'solid-js';

import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { toNumber } from 'panel/helpers/form';
import intl from 'panel/common/intl';
import { FILTER_INTERVALS } from './FilterUpdateModal';

export const FILTER_INTERVAL_RANGE = {
    MIN: 1,
    MAX: 8760,
};

type Props = {
    value: number | null | undefined;
    onChange: (value: number | null) => void;
    processing: boolean;
    intervalValue: number;
};

export const FilterIntervalInput = (props: Props) => {
    let inputRef: HTMLInputElement | undefined;
    const [prevInterval, setPrevInterval] = createSignal(untrack(() => props.intervalValue));

    createEffect(() => {
        const wasCustom = prevInterval() === -1;
        const isCustom = props.intervalValue === -1;

        setPrevInterval(props.intervalValue);

        if (!wasCustom && isCustom && !props.processing) {
            inputRef?.focus({ preventScroll: true });
        }
    });

    const handleChange = (e: Event) => {
        const target = e.target as HTMLInputElement;
        const { value } = target;
        props.onChange(toNumber(value));
    };

    const handleBlur = () => {
        // Validation on blur if needed
    };

    return (
        <div class={theme.form.input}>
            <Input
                ref={(el) => (inputRef = el)}
                id="filters_config_custom_interval"
                label={intl.getMessage('update_filters_custom_hours')}
                placeholder={intl.getMessage('settings_rotation_placeholder')}
                value={props.value ?? ''}
                onChange={handleChange}
                onBlur={handleBlur}
                disabled={props.processing || props.intervalValue !== FILTER_INTERVALS.CUSTOM}
                errorMessage={
                    props.intervalValue === -1 &&
                    props.value !== null &&
                    props.value !== undefined &&
                    (props.value < FILTER_INTERVAL_RANGE.MIN ||
                        props.value > FILTER_INTERVAL_RANGE.MAX)
                        ? intl.getMessage('form_error_range', {
                              min: FILTER_INTERVAL_RANGE.MIN,
                              max: FILTER_INTERVAL_RANGE.MAX,
                          })
                        : undefined
                }
                min={FILTER_INTERVAL_RANGE.MIN}
                max={FILTER_INTERVAL_RANGE.MAX}
                type="number"
            />
        </div>
    );
};
