import React from 'react';
import { components, NoticeProps } from 'react-select';
import intl from 'panel/common/intl';
import cn from 'clsx';
import theme from 'panel/lib/theme';

export const CustomNoOptionsMessage = <OptionType, IsMulti extends boolean>(
    props: NoticeProps<OptionType, IsMulti>,
) => (
    <components.NoOptionsMessage {...props}>
        <div className={cn(theme.select.empty, theme.text.t2)}>
            {intl.getMessage('nothing_found')}
        </div>
    </components.NoOptionsMessage>
);
