import React, { type ReactNode } from 'react';
import RCDialog from 'rc-dialog';
import cn from 'clsx';

import { Icon } from 'panel/common/ui';
import theme from 'panel/lib/theme';

import './Dialog.pcss';

type Props = {
    visible: boolean;
    mask?: boolean;
    className?: string;
    onClose: () => void;
    title?: ReactNode;
    dark?: boolean;
    minHeight?: boolean;
    wrapClassName?: string;
    children?: ReactNode;
};

export const Dialog = ({
    children,
    className,
    onClose,
    visible,
    title,
    mask = true,
    dark,
    minHeight,
    wrapClassName,
}: Props) => {
    const dialogClass = cn(
        {
            'dark-mode': dark,
            'min-height': minHeight,
        },
        className,
    );
    return (
        <RCDialog
            title={title}
            visible={visible}
            mask={mask}
            className={dialogClass}
            onClose={onClose}
            closeIcon={<Icon className={theme.dialog.close} icon="cross" />}
            classNames={{
                wrapper: wrapClassName,
            }}>
            {children}
        </RCDialog>
    );
};
