import { type JSX } from 'solid-js';
import cn from 'clsx';

import { Tooltip } from 'panel/common/ui/Tooltip';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import s from './styles.module.pcss';

type Props = {
    text: JSX.Element;
    menuSize?: 'small' | 'large';
    spacing?: boolean;
    menuClass?: string;
    overlayClass?: string;
    position?: 'bottomLeft' | 'bottomRight' | 'bottom';
};

export const FaqTooltip = (props: Props) => {
    const position = () => props.position ?? 'bottomLeft';

    return (
        <Tooltip
            overlayClass={cn(s.overlay_mobile, props.overlayClass)}
            content={
                <div
                    class={cn(theme.dropdown.menu, s.menu, props.menuClass, {
                        [s.menu_large]: props.menuSize === 'large',
                        [s.menu_spacing]: props.spacing,
                    })}
                >
                    {props.text}
                </div>
            }
            class={s.dropdown}
            position={position() as any}
        >
            <div class={s.trigger} onPointerDown={(e: PointerEvent) => e.stopPropagation()}>
                <Icon icon="faq" class={s.icon} />
            </div>
        </Tooltip>
    );
};
