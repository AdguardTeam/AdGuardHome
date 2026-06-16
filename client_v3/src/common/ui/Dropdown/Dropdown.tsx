import { type JSX, createSignal, createEffect, onCleanup, Show } from 'solid-js';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import Popover from '@corvu/popover';

import './Dropdown.pcss';
import s from './Dropdown.module.pcss';

const TIMEOUT_HIDE_TOOLTIP = 1000;

type Props = {
    overlayClass?: string;
    menu: JSX.Element;
    position?: 'bottomRight' | 'bottomLeft' | 'topRight' | 'topLeft' | 'top';
    trigger: 'click' | 'hover';
    noIcon?: true;
    iconClass?: string;
    class?: string;
    openClass?: string;
    open?: boolean;
    onOpenChange?: (e: boolean) => void;
    widthAuto?: boolean;
    flex?: boolean;
    minOverlayWidthMatchTrigger?: boolean;
    flexWrapper?: boolean;
    childrenClass?: string;
    wrapClass?: string;
    children?: JSX.Element;
    isSelect?: boolean;
    disableAnimation?: boolean;
    disabled?: boolean;
    autoClose?: boolean;
};

export const Dropdown = (props: Props) => {
    let timer: ReturnType<typeof setTimeout> | null = null;
    const [visible, setVisible] = createSignal(!!props.open);

    const onVisibleChange = (e: boolean) => {
        if (props.disabled) {
            return;
        }

        props.onOpenChange?.(e);
        setVisible(e);
    };

    createEffect(() => {
        if (typeof props.open === 'boolean') {
            setVisible(props.open);
        }
    });

    onCleanup(() => {
        setVisible(false);
        if (timer) {
            clearTimeout(timer);
        }
    });

    const handleOverlayClick = () => {
        if (!props.autoClose) {
            return;
        }

        if (timer) {
            clearTimeout(timer);
        }
        timer = setTimeout(() => {
            setVisible(false);
            onVisibleChange(false);
        }, TIMEOUT_HIDE_TOOLTIP);
    };

    return (
        <Popover
            open={visible()}
            onOpenChange={onVisibleChange}
            placement={props.position === 'topRight' || props.position === 'topLeft' ? 'top' : 'bottom'}
            floatingOptions={{
                offset: 4,
            }}
        >
            <Popover.Anchor>
                <div
                    class={cn(
                        props.class,
                        s.wrapper,
                        {
                            [s.open]: props.flex,
                            [s.disabled]: props.disabled,
                        },
                        visible() && props.openClass ? props.openClass : null,
                        props.wrapClass,
                    )}
                >
                    <div
                        class={cn(props.childrenClass, { [s.wrapper]: props.flexWrapper })}
                        onClick={props.trigger === 'click' ? () => onVisibleChange(!visible()) : undefined}
                        onMouseEnter={props.trigger === 'hover' ? () => onVisibleChange(true) : undefined}
                        onMouseLeave={props.trigger === 'hover' ? () => onVisibleChange(false) : undefined}
                    >
                        {props.children}
                    </div>
                    <Show when={!props.noIcon}>
                        <Icon
                            class={cn(s.arrow, props.iconClass, { [s.active]: visible() })}
                            icon="arrow_bottom"
                        />
                    </Show>
                </div>
            </Popover.Anchor>
            <Popover.Portal>
                <Popover.Content
                    class={cn(s.overlay, props.overlayClass, {
                        [s.widthAuto]: props.widthAuto,
                        [s.selectOverlay]: props.isSelect,
                    })}
                    onClick={handleOverlayClick}
                >
                    {props.menu}
                </Popover.Content>
            </Popover.Portal>
        </Popover>
    );
};
