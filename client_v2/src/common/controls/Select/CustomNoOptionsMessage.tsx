import React from 'react';
import { components, NoticeProps } from 'react-select';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

export const CustomNoOptionsMessage = <OptionType, IsMulti extends boolean>(
    props: NoticeProps<OptionType, IsMulti>,
) => (
    <components.NoOptionsMessage {...props}>
        <span className={theme.select.empty}>{intl.getMessage('not_found')}</span>
    </components.NoOptionsMessage>
);
