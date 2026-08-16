import cn from 'clsx';
import { IconType } from 'panel/common/ui/Icons';

import s from './Icon.module.pcss';

export type IconColor = 'green' | 'gray' | 'red' | 'black';

type Props = {
    icon: IconType;
    color?: IconColor;
    class?: string;
    onClick?: (e: MouseEvent) => void;
};

export const Icon = (props: Props) => {
    const iconClass = () => cn(s.icon, s[props.color!], props.class);

    return (
        <svg class={iconClass()} onClick={(e) => props.onClick?.(e)}>
            <use href={`#${props.icon}`} />
        </svg>
    );
};

export type { IconType } from 'panel/common/ui/Icons';
