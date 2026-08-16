import { type JSX, Show } from 'solid-js';
import cn from 'clsx';
import { Button, type ButtonProps } from 'panel/common/ui/Button';
import { Dialog } from 'panel/common/ui/Dialog';
import theme from 'panel/lib/theme';

import s from './ConfirmDialog.module.pcss';

type Props = {
    onClose: () => void;
    title?: JSX.Element;
    text?: JSX.Element;
    buttonText: string;
    cancelText: string;
    onConfirm?: () => void;
    submitDisabled?: boolean;
    buttonVariant?: ButtonProps['variant'];
    submitId?: string;
    cancelId?: string;
    customFooter?: JSX.Element;
    wrapClass?: string;
    submitTestId?: string;
    cancelTestId?: string;
};

export const ConfirmDialog = (props: Props) => (
    <Dialog
        mask
        visible
        title={props.title}
        onClose={props.onClose}
        class={s.сonfirmDialogClass}
        wrapClass={cn('rc-dialog-update', props.wrapClass)}
    >
        <Show when={props.text}>
            <div class={theme.dialog.body}>{props.text}</div>
        </Show>

        <Show
            when={props.customFooter}
            fallback={
                <div class={theme.dialog.footer}>
                    <Button
                        id={props.submitId}
                        data-testid={props.submitTestId}
                        variant={props.buttonVariant ?? 'primary'}
                        size="small"
                        onClick={props.onConfirm}
                        class={theme.dialog.button}
                        disabled={props.submitDisabled ?? false}
                    >
                        {props.buttonText}
                    </Button>

                    <Button
                        id={props.cancelId}
                        data-testid={props.cancelTestId}
                        variant="secondary"
                        size="small"
                        onClick={props.onClose}
                        class={theme.dialog.button}
                    >
                        {props.cancelText}
                    </Button>
                </div>
            }
        >
            {props.customFooter}
        </Show>
    </Dialog>
);
