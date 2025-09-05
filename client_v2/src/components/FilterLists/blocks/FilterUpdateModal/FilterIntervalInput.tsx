import React, { useEffect, useRef } from 'react';
import { Controller, type Control, type Path } from 'react-hook-form';

import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { toNumber } from 'panel/helpers/form';
import intl from 'panel/common/intl';
import { FILTER_INTERVALS } from './FilterUpdateModal';

const FILTER_INTERVAL_RANGE = {
    MIN: 1,
    MAX: 8760,
};

type Props<TFormValues extends { customInterval?: number | null }> = {
    control: Control<TFormValues>;
    processing: boolean;
    intervalValue: number;
};

export const FilterIntervalInput = <TFormValues extends { customInterval?: number | null }>({
    control,
    processing,
    intervalValue,
}: Props<TFormValues>) => {
    const inputRef = useRef<HTMLInputElement>(null);
    const prevIntervalRef = useRef(intervalValue);

    useEffect(() => {
        const wasCustom = prevIntervalRef.current === -1;
        const isCustom = intervalValue === -1;

        prevIntervalRef.current = intervalValue;

        if (!wasCustom && isCustom && !processing) {
            inputRef.current?.focus({ preventScroll: true });
        }
    }, [intervalValue, processing]);

    return (
        <Controller
            name={'customInterval' as Path<TFormValues>}
            control={control}
            rules={{
                validate: (value) => {
                    if (intervalValue === -1) {
                        if (!value || value < FILTER_INTERVAL_RANGE.MIN || value > FILTER_INTERVAL_RANGE.MAX) {
                            return intl.getMessage('form_error_range', {
                                min: FILTER_INTERVAL_RANGE.MIN,
                                max: FILTER_INTERVAL_RANGE.MAX,
                            });
                        }
                    }
                    return true;
                },
            }}
            render={({ field, fieldState }) => (
                <div className={theme.form.input}>
                    <Input
                        ref={inputRef}
                        id="filters_config_custom_interval"
                        label={intl.getMessage('update_filters_custom_hours')}
                        placeholder={intl.getMessage('settings_rotation_placeholder')}
                        value={field.value ?? ''}
                        onChange={(e) => {
                            const { value } = e.target;
                            field.onChange(toNumber(value));
                        }}
                        onBlur={field.onBlur}
                        disabled={processing || intervalValue !== FILTER_INTERVALS.CUSTOM}
                        errorMessage={fieldState.error?.message}
                        min={FILTER_INTERVAL_RANGE.MIN}
                        max={FILTER_INTERVAL_RANGE.MAX}
                        type="number"
                    />
                </div>
            )}
        />
    );
};
