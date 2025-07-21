import React from 'react';
import cn from 'clsx';
import { Icon } from 'panel/common/ui';
import theme from 'panel/lib/theme';

type Props = {
    isSelected: boolean;
    isMulti: boolean;
};

export const CustomOptionIcon = ({ isSelected, isMulti }: Props) => {
    if (isMulti) {
        return (
            <Icon
                icon={isSelected ? 'checkbox_checked' : 'checkbox_unchecked'}
                className={cn(theme.select.check, { [theme.select.check_active]: isSelected })}
            />
        );
    }

    return <Icon icon={isSelected ? 'check' : 'dot'} />;
};
