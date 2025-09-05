import React, { ReactNode } from 'react';
import cn from 'clsx';
import { Button, ButtonProps } from 'panel/common/ui/Button';
import { Dialog } from 'panel/common/ui/Dialog';
import theme from 'panel/lib/theme';

import s from './ConfirmDialog.module.pcss';

type Props = {
    onClose: () => void;
    title?: ReactNode;
    text?: ReactNode;
    buttonText: string;
    cancelText: string;
    onConfirm?: () => void;
    submitDisabled?: boolean;
    buttonVariant?: ButtonProps['variant'];
    submitId?: string;
    cancelId?: string;
    customFooter?: ReactNode;
    wrapClassName?: string;
};

export const ConfirmDialog = ({
    title,
    text,
    buttonText,
    onClose,
    onConfirm,
    submitDisabled = false,
    buttonVariant = 'primary',
    submitId,
    cancelId,
    customFooter,
    wrapClassName,
    cancelText,
}: Props) => (
    <Dialog
        mask
        visible
        title={title}
        onClose={onClose}
        className={s.сonfirmDialogClass}
        wrapClassName={cn('rc-dialog-update', wrapClassName)}>
        {text && <div className={theme.dialog.body}>{text}</div>}

        {customFooter || (
            <div className={theme.dialog.footer}>
                <Button
                    id={submitId}
                    variant={buttonVariant}
                    size="small"
                    onClick={onConfirm}
                    className={theme.dialog.button}
                    disabled={submitDisabled}>
                    {buttonText}
                </Button>

                <Button
                    id={cancelId}
                    variant="secondary"
                    size="small"
                    onClick={onClose}
                    className={theme.dialog.button}>
                    {cancelText}
                </Button>
            </div>
        )}
    </Dialog>
);
