import cn from 'clsx';
import { Icon, type IconColor, type IconType } from 'panel/common/ui/Icon';

import s from './Loader.module.pcss';

type Props = {
    color?: IconColor;
    class?: string;
    overlay?: boolean;
    overlayClass?: string;
    icon?: IconType;
};

export const Loader = (props: Props) => (
    <div class={cn({ [s.overlayWrapper]: props.overlay }, props.overlayClass)}>
        <div class={cn({ [s.overlay]: props.overlay })}>
            <Icon
                color={props.color}
                class={cn(s.loader, props.class)}
                icon={props.icon || 'loader'}
            />
        </div>
    </div>
);
