import React, { ReactNode } from 'react';
import cn from 'clsx';

import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import s from './styles.module.pcss';

type Props = {
    text: ReactNode;
    menuSize?: 'small' | 'large';
};

export const FaqTooltip = ({ text, menuSize = 'small' }: Props) => {
    return (
        <Dropdown
            trigger="hover"
            menu={
                <div
                    className={cn(theme.dropdown.menu, s.menu, {
                        [s.menu_large]: menuSize === 'large',
                    })}>
                    {text}
                </div>
            }
            className={s.dropdown}
            position="bottomLeft"
            noIcon>
            <div className={s.trigger}>
                <Icon icon="faq" className={s.icon} />
            </div>
        </Dropdown>
    );
};
