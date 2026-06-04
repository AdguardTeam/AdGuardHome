import React from 'react';
import cn from 'clsx';
import { ClearIndicatorProps } from 'react-select';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

export const CustomClearIndicator = <
    OptionType extends Record<string, any> = any,
    IsMulti extends boolean = false,
>(
    props: ClearIndicatorProps<OptionType, IsMulti>,
) => {
    const { hasValue, innerProps, selectProps } = props;

    if (!hasValue) {
        return null;
    }

    const noDropdownIndicator = selectProps.isMulti && hasValue;

    return (
        <div
            role="button"
            tabIndex={0}
            className={cn(
                theme.select.clearIndicator,
                noDropdownIndicator && theme.select.clearIndicatorEnd,
            )}
            {...innerProps}
        >
            <Icon icon="cross" className={theme.select.clearIcon} />
        </div>
    );
};
