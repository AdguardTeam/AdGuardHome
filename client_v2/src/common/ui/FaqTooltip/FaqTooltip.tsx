import React from 'react';
import type { ReactNode } from 'react';
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
    label?: ReactNode;
};

export const FaqTooltip = ({
    text,
    menuSize = 'small',
    spacing = false,
    menuClassName,
    overlayClassName,
    position: positionProp,
    label,
}: Props) => {
    const isMobile = useIsMobile();

    const getDefaultPosition = () => {
        if (isMobile) {
            return label ? 'bottomLeft' : 'bottom';
        }

        return 'bottomLeft';
    };

    const position = positionProp ?? getDefaultPosition();

    const trigger = label ? (
        <span className={s.labelTrigger}>
            {label}
            <Icon icon="faq" className={s.icon} />
        </span>
    ) : (
        <Icon icon="faq" className={s.icon} />
    );

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
                {trigger}
            </div>
        </Dropdown>
    );
};
