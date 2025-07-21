import React from 'react';
import { GroupBase, OptionProps } from 'react-select';
import cn from 'clsx';
import { Icon } from 'panel/common/ui';
import theme from 'panel/lib/theme';

export const CustomOption = <
    OptionType extends Record<string, any>,
    IsMulti extends boolean,
    Group extends GroupBase<OptionType>,
>({
    data,
    isSelected,
    selectOption,
}: OptionProps<OptionType, IsMulti, Group>) => (
    <div className={cn(theme.select.option, theme.select.option_check)} onClick={() => selectOption(data)}>
        <Icon icon={isSelected ? 'check' : 'dot'} className={theme.select.icon} />
        <span className={theme.common.textOverflow}>{data.label}</span>
    </div>
);
