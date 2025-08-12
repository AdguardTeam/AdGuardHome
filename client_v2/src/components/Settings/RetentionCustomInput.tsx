import React, { useEffect, useRef } from 'react';
import { Controller, type Control, type Path } from 'react-hook-form';

import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { RETENTION_CUSTOM, RETENTION_RANGE } from 'panel/helpers/constants';
import { toNumber } from 'panel/helpers/form';

type Props<TFormValues extends { customInterval?: number | null }> = {
    control: Control<TFormValues>;
    processing: boolean;
    intervalValue: number;
    intervals: number[];
    inputId: string;
    inputLabel: string;
    placeholder: string;
};

export const RetentionCustomInput = <TFormValues extends { customInterval?: number | null }>({
    control,
    processing,
    intervalValue,
    intervals,
    inputId,
    inputLabel,
    placeholder,
}: Props<TFormValues>) => {
    const inputRef = useRef<HTMLInputElement>(null);
    const prevIntervalRef = useRef(intervalValue);

    useEffect(() => {
        const wasCustom = prevIntervalRef.current === RETENTION_CUSTOM;
        const isCustom = intervalValue === RETENTION_CUSTOM;

        prevIntervalRef.current = intervalValue;

        if (!wasCustom && isCustom && !processing) {
            inputRef.current?.focus({ preventScroll: true });
        }
    }, [intervalValue, processing]);

    return (
        <Controller
            name={'customInterval' as Path<TFormValues>}
            control={control}
            render={({ field, fieldState }) => (
                <div className={theme.form.input}>
                    <Input
                        ref={inputRef}
                        id={inputId}
                        label={inputLabel}
                        placeholder={placeholder}
                        value={field.value ?? ''}
                        onChange={(e) => {
                            const { value } = e.target;
                            field.onChange(toNumber(value));
                        }}
                        onBlur={field.onBlur}
                        disabled={processing || intervals.includes(intervalValue)}
                        error={!!fieldState.error}
                        min={RETENTION_RANGE.MIN}
                        max={RETENTION_RANGE.MAX}
                    />
                </div>
            )}
        />
    );
};
