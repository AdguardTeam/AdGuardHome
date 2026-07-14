import { createEffect, untrack } from 'solid-js';

import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { RETENTION_CUSTOM } from 'panel/helpers/constants';
import { toNumber } from 'panel/helpers/form';

type Props = {
    value: number | null | undefined;
    onChange: (value: number | null) => void;
    onBlur?: () => void;
    processing: boolean;
    intervalValue: number;
    intervals: number[];
    inputId: string;
    inputLabel: string;
    placeholder: string;
    error?: string;
};

export const RetentionCustomInput = (props: Props) => {
    let inputRef: HTMLInputElement | undefined;
    let prevInterval = untrack(() => props.intervalValue);

    createEffect(() => {
        const intervalValue = props.intervalValue;
        const wasCustom = prevInterval === RETENTION_CUSTOM;
        const isCustom = intervalValue === RETENTION_CUSTOM;

        prevInterval = intervalValue;

        if (!wasCustom && isCustom && !props.processing) {
            inputRef?.focus({ preventScroll: true });
        }
    });

    return (
        <div class={theme.form.input}>
            <Input
                ref={inputRef}
                id={props.inputId}
                type="number"
                label={props.inputLabel}
                placeholder={props.placeholder}
                value={props.value ?? ''}
                onChange={(e: Event) => {
                    const { value } = e.target as HTMLInputElement;
                    props.onChange(toNumber(value));
                }}
                onBlur={() => props.onBlur?.()}
                disabled={props.processing || props.intervals.includes(props.intervalValue)}
                error={!!props.error}
                errorMessage={props.error}
                size="large"
            />
        </div>
    );
};
