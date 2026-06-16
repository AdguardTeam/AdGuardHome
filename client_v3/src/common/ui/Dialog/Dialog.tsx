import { type JSX, Show } from 'solid-js';
import DialogPrimitive from '@corvu/dialog';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import './Dialog.pcss';

type Props = {
    visible: boolean;
    mask?: boolean;
    class?: string;
    onClose: () => void;
    title?: JSX.Element;
    wrapClass?: string;
    children?: JSX.Element;
};

export const Dialog = (props: Props) => {
    return (
        <DialogPrimitive
            open={props.visible}
            onOpenChange={(open) => {
                if (!open) props.onClose();
            }}
        >
            <Show when={props.mask !== false}>
                <DialogPrimitive.Overlay class="dialog-overlay" />
            </Show>
            <DialogPrimitive.Portal>
                <DialogPrimitive.Content
                    class={`dialog-content ${props.class || ''} ${props.wrapClass || ''}`}
                >
                    <Show when={props.title}>
                        <div class="dialog-header">
                            <DialogPrimitive.Label class="dialog-title">
                                {props.title}
                            </DialogPrimitive.Label>
                            <DialogPrimitive.Close class="dialog-close-button">
                                <Icon class={theme.dialog.close} icon="cross" />
                            </DialogPrimitive.Close>
                        </div>
                    </Show>
                    <div class="dialog-body">{props.children}</div>
                </DialogPrimitive.Content>
            </DialogPrimitive.Portal>
        </DialogPrimitive>
    );
};
