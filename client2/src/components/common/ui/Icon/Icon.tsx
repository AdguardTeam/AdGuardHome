import React, { FC } from 'react';
import cn from 'classnames';
import { IconType } from 'Common/ui/Icons';

import s from './Icon.module.pcss';

interface IconProps {
    icon: IconType;
    color?: string;
    className?: string;
    onClick?: () => void;
}

const Icon: FC<IconProps> = ({ icon, color, className, onClick }) => {
    const iconClass = cn(s.icon, color, className);

    return (
        <svg className={iconClass} onClick={onClick}>
            <use xlinkHref={`#${icon}`} />
        </svg>
    );
};

export default Icon;
export { IconType } from 'Common/ui/Icons';
