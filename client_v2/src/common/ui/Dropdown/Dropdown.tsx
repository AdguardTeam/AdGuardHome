import { type JSX, createSignal, createEffect, onCleanup, Show, untrack } from 'solid-js';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import { Popover } from '@ark-ui/solid';

import './Dropdown.pcss';
import s from './Dropdown.module.pcss';

type Props = {
    overlayClass?: string;
    menu: JSX.Element;
    position?: 'bottomRight' | 'bottomLeft' | 'topRight' | 'topLeft' | 'top';
    noIcon?: true;
    iconClass?: string;
    class?: string;
    open?: boolean;
    onOpenChange?: (e: boolean) => void;
    widthAuto?: boolean;
    flex?: boolean;
    flexWrapper?: boolean;
    childrenClass?: string;
    wrapClass?: string;
    children?: JSX.Element;
    isSelect?: boolean;
    disabled?: boolean;
};

export const Dropdown = (props: Props) => {
    const [visible, setVisible] = createSignal(!!untrack(() => props.open));

    const onVisibleChange = (details: { open: boolean }) => {
        if (props.disabled) {
            return;
        }

        props.onOpenChange?.(details.open);
        setVisible(details.open);
    };

    createEffect(() => {
        if (typeof props.open === 'boolean') {
            setVisible(props.open);
        }
    });

    onCleanup(() => {
        setVisible(false);
    });

    // Ark UI uses floating-ui placement tokens. corvu collapsed every position
    // to plain `bottom`/`top` (ignoring left/right), so mapping `bottomRight` ->
    // `bottom-end` is an intentional behavior change that honors the `position`
    // prop. `flip` keeps the popover on-screen near viewport edges.
    const placement = () => {
        const position = untrack(() => props.position);
        switch (position) {
            case 'topRight':
                return 'top-end';
            case 'topLeft':
                return 'top-start';
            case 'bottomRight':
                return 'bottom-end';
            case 'bottomLeft':
                return 'bottom-start';
            case 'top':
                return 'top';
            default:
                return 'bottom-end';
        }
    };

    return (
        <Popover.Root
            open={visible()}
            onOpenChange={onVisibleChange}
            positioning={{
                placement: placement(),
                gutter: 4,
                flip: true,
            }}
            closeOnInteractOutside={true}
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
                        props.wrapClass,
                    )}
                >
                    <Popover.Trigger
                        asChild={(triggerProps) => (
                            <div
                                {...triggerProps}
                                class={cn(s.trigger, props.childrenClass, {
                                    [s.flexWrapper]: props.flexWrapper,
                                })}
                            >
                                {props.children}
                                <Show when={!props.noIcon}>
                                    <Icon
                                        aria-hidden="true"
                                        class={cn(s.arrow, props.iconClass, {
                                            [s.active]: visible(),
                                        })}
                                        icon="arrow_bottom"
                                    />
                                </Show>
                            </div>
                        )}
                    />
                </div>
            </Popover.Anchor>
            <Popover.Positioner>
                <Popover.Content
                    class={cn(s.overlay, props.overlayClass, {
                        [s.widthAuto]: props.widthAuto,
                        [s.selectOverlay]: props.isSelect,
                    })}
                >
                    {props.menu}
                </Popover.Content>
            </Popover.Positioner>
        </Popover.Root>
    );
};
