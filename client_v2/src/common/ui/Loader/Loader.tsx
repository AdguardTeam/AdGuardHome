import React from 'react';
import cn from 'clsx';
import { Icon, IconColor, IconType } from 'panel/common/ui/Icon';

import s from './Loader.module.pcss';

type Props = {
    color?: IconColor;
    className?: string;
    overlay?: boolean;
    overlayClassName?: string;
    icon?: IconType;
};

export const Loader = ({ color, className, overlay, overlayClassName, icon = 'loader' }: Props) => (
    <div className={cn({ [s.overlayWrapper]: overlay }, overlayClassName)}>
        <div className={cn({ [s.overlay]: overlay })}>
            <Icon color={color} className={cn(s.loader, className)} icon={icon} />
        </div>
    </div>
);
