import React, { MouseEvent } from 'react';
import cn from 'clsx';
import { IconType } from 'panel/common/ui/Icons';

import s from './Icon.module.pcss';

export type IconColor = 'green' | 'gray' | 'red' | 'black';

type Props = {
    icon: IconType;
    color?: IconColor;
    className?: string;
    onClick?: (e: MouseEvent) => void;
};

export const Icon = ({ icon, color, className, onClick }: Props) => {
    const iconClass = cn(s.icon, s[color], className);

    return (
        <svg className={iconClass} onClick={onClick}>
            <use xlinkHref={`#${icon}`} />
        </svg>
    );
};

export type { IconType } from 'panel/common/ui/Icons';
