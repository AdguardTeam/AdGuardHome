import React from 'react';
import { GroupBase, OptionProps } from 'react-select';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

type CustomOptionProps<
    OptionType extends Record<string, any>,
    IsMulti extends boolean,
    Group extends GroupBase<OptionType>,
> = OptionProps<OptionType, IsMulti, Group> & {
    showIcon?: boolean;
};

export const CustomOption = <
    OptionType extends Record<string, any>,
    IsMulti extends boolean,
    Group extends GroupBase<OptionType>,
>({
    data,
    isDisabled,
    isSelected,
    selectOption,
    showIcon = true,
}: CustomOptionProps<OptionType, IsMulti, Group>) => (
    <div
        className={cn(
            theme.select.option,
            theme.select.option_check,
            {
                [theme.select.option_disabled]: isDisabled,
                [theme.select.option_selected]: isSelected,
            },
        )}
        onClick={isDisabled ? undefined : () => selectOption(data)}
        aria-disabled={isDisabled}
    >
        {showIcon && <Icon icon={isSelected ? 'check' : 'dot'} className={theme.select.icon} />}
        <span className={theme.common.textOverflow}>{data.label}</span>
    </div>
);
