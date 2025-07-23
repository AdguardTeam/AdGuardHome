import React, { type ReactNode } from 'react';
import RCDialog from 'rc-dialog';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import './Dialog.pcss';

type Props = {
    visible: boolean;
    mask?: boolean;
    className?: string;
    onClose: () => void;
    title?: ReactNode;
    wrapClassName?: string;
    children?: ReactNode;
};

export const Dialog = ({ children, className, onClose, visible, title, mask = true, wrapClassName }: Props) => {
    return (
        <RCDialog
            title={title}
            visible={visible}
            mask={mask}
            className={className}
            onClose={onClose}
            closeIcon={<Icon className={theme.dialog.close} icon="cross" />}
            classNames={{
                wrapper: wrapClassName,
            }}>
            {children}
        </RCDialog>
    );
};
