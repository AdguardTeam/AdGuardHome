import React from 'react';
import { DropdownIndicatorProps } from 'react-select';
import cn from 'clsx';
import { Icon } from 'panel/common/ui';
import theme from 'panel/lib/theme';

export const CustomDropdownIndicator = <OptionType extends Record<string, any> = any, IsMulti extends boolean = false>(
    props: DropdownIndicatorProps<OptionType, IsMulti>,
) => {
    const { selectProps } = props;
    const isMenuOpen = selectProps.menuIsOpen;

    return (
        <div className={theme.select.dropdownIndicator}>
            <Icon
                icon="arrow_bottom"
                className={cn(theme.select.dropdownIcon, {
                    [theme.select.dropdownIconOpen]: isMenuOpen,
                })}
            />
        </div>
    );
};
