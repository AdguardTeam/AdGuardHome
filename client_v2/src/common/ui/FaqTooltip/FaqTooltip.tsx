import React, { ReactNode } from 'react';
import cn from 'clsx';

import { Dropdown } from 'panel/common/ui/Dropdown';
import { useIsMobile } from 'panel/hooks/useIsMobile';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import s from './styles.module.pcss';

type Props = {
    text: ReactNode;
    menuSize?: 'small' | 'large';
    spacing?: boolean;
    menuClassName?: string;
    overlayClassName?: string;
    position?: 'bottomLeft' | 'bottomRight' | 'bottom';
};

export const FaqTooltip = ({
    text,
    menuSize = 'small',
    spacing = false,
    menuClassName,
    overlayClassName,
    position: positionProp,
}: Props) => {
    const isMobile = useIsMobile();

    const currentPosition = isMobile ? 'bottom' : 'bottomLeft';
    const position = positionProp ?? currentPosition;

    return (
        <Dropdown
            trigger={isMobile ? 'click' : 'hover'}
            overlayClassName={cn(s.overlay_mobile, overlayClassName)}
            menu={
                <div
                    className={cn(theme.dropdown.menu, s.menu, menuClassName, {
                        [s.menu_large]: menuSize === 'large',
                        [s.menu_spacing]: spacing,
                    })}
                >
                    {text}
                </div>
            }
            className={s.dropdown}
            position={position}
            noIcon
        >
            <div className={s.trigger} onPointerDown={(e) => e.stopPropagation()}>
                <Icon icon="faq" className={s.icon} />
            </div>
        </Dropdown>
    );
};
