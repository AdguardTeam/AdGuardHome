import { type JSX, Show } from 'solid-js';
import { Dialog as ArkDialog } from '@ark-ui/solid';

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
        <ArkDialog.Root
            open={props.visible}
            onOpenChange={(details) => {
                if (!details.open) props.onClose();
            }}
            closeOnInteractOutside={true}
        >
            <Show when={props.mask !== false}>
                <ArkDialog.Backdrop class="dialog-overlay" />
            </Show>
            <ArkDialog.Positioner class={`rc-dialog-wrap ${props.wrapClass || ''}`}>
                <ArkDialog.Content class={`dialog-content ${props.class || ''}`}>
                    <Show when={props.title}>
                        <div class="dialog-header">
                            <ArkDialog.Title class="dialog-title">{props.title}</ArkDialog.Title>
                            <ArkDialog.CloseTrigger class="dialog-close-button">
                                <Icon class={theme.dialog.close} icon="cross" />
                            </ArkDialog.CloseTrigger>
                        </div>
                    </Show>
                    <div class="dialog-body">{props.children}</div>
                </ArkDialog.Content>
            </ArkDialog.Positioner>
        </ArkDialog.Root>
    );
};
