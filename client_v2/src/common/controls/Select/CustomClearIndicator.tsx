import React from 'react';
import { ClearIndicatorProps } from 'react-select';
import { Icon } from 'panel/common/ui';
import theme from 'panel/lib/theme';

export const CustomClearIndicator = <OptionType extends Record<string, any> = any, IsMulti extends boolean = false>(
    props: ClearIndicatorProps<OptionType, IsMulti>,
) => {
    const { hasValue, innerProps } = props;

    if (!hasValue) {
        return null;
    }

    return (
        <div className={theme.select.clearIndicator} {...innerProps}>
            <Icon icon="cross" className={theme.select.clearIcon} />
        </div>
    );
};
