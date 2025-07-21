import React from 'react';
import { components, GroupBase, NoticeProps } from 'react-select';
import theme from 'panel/lib/theme';

import { Loader } from 'panel/common/ui';

export const CustomLoadingMessage = <
    Option,
    IsMulti extends boolean = false,
    Group extends GroupBase<Option> = GroupBase<Option>,
>(
    props: NoticeProps<Option, IsMulti, Group>,
) => (
    <components.LoadingMessage {...props}>
        <Loader icon="loader" className={theme.select.menuLoader} overlayClassName={theme.select.menuLoaderOverlay} />
    </components.LoadingMessage>
);
