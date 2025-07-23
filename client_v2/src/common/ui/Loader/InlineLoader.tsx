import React from 'react';
import cn from 'clsx';
import { Icon, IconType } from 'panel/common/ui/Icon';

import s from './Loader.module.pcss';

type Props = {
    className?: string;
    icon?: IconType;
};

export const InlineLoader = ({ className, icon = 'loader' }: Props) => (
    <Icon className={cn(s.loader, className)} icon={icon} />
);
