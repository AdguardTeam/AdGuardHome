import React, { ReactNode } from 'react';
import cn from 'clsx';

import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import s from './styles.module.pcss';

type Props = {
    text: ReactNode;
    menuSize?: 'small' | 'large';
    spacing?: boolean;
    menuClassName?: string;
    overlayClassName?: string;
};

export const FaqTooltip = ({ text, menuSize = 'small', spacing = false, menuClassName, overlayClassName }: Props) => {
    return (
        <Dropdown
            trigger="hover"
            overlayClassName={overlayClassName}
            menu={
                <div
                    className={cn(theme.dropdown.menu, s.menu, menuClassName, {
                        [s.menu_large]: menuSize === 'large',
                        [s.menu_spacing]: spacing,
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
