import React, { MouseEvent } from 'react';
import cn from 'clsx';
import { IconType } from 'panel/common/ui/Icons';

import s from './Icon.module.pcss';

interface IconProps {
    icon: IconType;
    color?: string;
    className?: string;
    small?: boolean;
    onClick?: (e: MouseEvent) => void;
}

export const Icon = ({ icon, color, className, onClick, small }: IconProps) => {
    const iconClass = cn(s.icon, color, className, { [s.smallIcon]: small });

    return (
        <svg className={iconClass} onClick={onClick}>
            <use xlinkHref={`#${icon}`} />
        </svg>
    );
};

export type { IconType } from 'panel/common/ui/Icons';
