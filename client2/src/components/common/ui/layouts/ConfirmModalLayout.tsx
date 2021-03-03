import React, { FC } from 'react';

import CommonModalLayout from './CommonModalLayout';

interface DeleteModalLayoutProps {
    visible: boolean;
    title: string;
    buttonText: string;
    onClose: () => void;
    onConfirm?: () => void;
}

const DeleteModalLayout: FC<DeleteModalLayoutProps> = ({
    visible,
    children,
    title,
    buttonText,
    onClose,
    onConfirm,
}) => {
    return (
        <CommonModalLayout
            visible={visible}
            title={title}
            buttonText={buttonText}
            onSubmit={onConfirm}
            onClose={onClose}
        >
            {children}
        </CommonModalLayout>
    );
};

export default DeleteModalLayout;
