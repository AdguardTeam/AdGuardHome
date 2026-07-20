import { type JSX, Show, createMemo, createSignal } from 'solid-js';
import cn from 'clsx';
import { Tooltip as ArkTooltip } from '@ark-ui/solid';

import { useMediaQuery } from 'panel/helpers/useMediaQuery';
import { useOutsideDismiss } from 'panel/hooks/useOutsideDismiss';

import './Tooltip.pcss';
import s from './Tooltip.module.pcss';

const SHOW_DELAY = 200;
const HIDE_DELAY = 300;

type Props = {
    content: JSX.Element;
    position?: 'bottomRight' | 'bottomLeft' | 'topRight' | 'topLeft' | 'top';
    overlayClass?: string;
    class?: string;
    disabled?: boolean;
    children?: JSX.Element;
};

export const Tooltip = (props: Props) => {
    const placement = createMemo(
        (): 'top-end' | 'top-start' | 'bottom-end' | 'bottom-start' | 'top' => {
            switch (props.position) {
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
        },
    );

    const positioning = createMemo(() => ({
        placement: placement(),
        gutter: 4,
        flip: true,
    }));

    // Touch-device detection via hover-media query.
    const isTouch = useMediaQuery('(hover: none)');

    // Managed open-state: hover-driven on desktop, tap-driven on touch.
    const [hoverOpen, setHoverOpen] = createSignal(false);
    const [tapOpen, setTapOpen] = createSignal(false);
    const open = () => hoverOpen() || tapOpen();

    let triggerRef: HTMLDivElement | undefined;
    let contentRef: HTMLDivElement | undefined;

    // On touch devices: tapping outside an open tooltip closes it.
    useOutsideDismiss(
        () => isTouch() && tapOpen(),
        () => setTapOpen(false),
        triggerRef,
        contentRef,
    );

    // Zag.js Tooltip's "disabled" prop only closes an already-open tooltip
    // when disabled changes to true; it does not prevent opening on hover.
    // Render children without the tooltip wrapper when disabled.
    return (
        <Show
            when={!props.disabled}
            fallback={<div class={cn(props.class, s.wrapper)}>{props.children}</div>}
        >
            <ArkTooltip.Root
                open={open()}
                onOpenChange={(details) => setHoverOpen(details.open)}
                openDelay={SHOW_DELAY}
                closeDelay={HIDE_DELAY}
                interactive
                positioning={positioning()}
                closeOnClick={false}
                closeOnPointerDown={false}
                closeOnScroll={false}
                closeOnEscape={false}
            >
                <ArkTooltip.Trigger
                    asChild={(triggerProps) => (
                        <div
                            {...triggerProps}
                            ref={(el) => {
                                triggerRef = el;
                            }}
                            class={cn(props.class, s.wrapper)}
                            onClick={() => {
                                if (isTouch()) {
                                    setTapOpen((v) => !v);
                                }
                            }}
                        >
                            {props.children}
                        </div>
                    )}
                />
                <ArkTooltip.Positioner>
                    <ArkTooltip.Content class={cn(s.overlay, props.overlayClass)}>
                        <div
                            ref={(el) => {
                                contentRef = el;
                            }}
                        >
                            {props.content}
                        </div>
                    </ArkTooltip.Content>
                </ArkTooltip.Positioner>
            </ArkTooltip.Root>
        </Show>
    );
};
