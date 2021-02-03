import React, { FC, useContext, useEffect } from 'react';
import { Modal, Button } from 'antd';
import cn from 'classnames';

import { Icon } from 'Common/ui';
import Store from 'Store';

interface CommonModalLayoutProps {
    visible: boolean;
    title: string;
    buttonText?: string;
    className?: string;
    width?: number;
    onClose: () => void;
    onSubmit?: () => void;
    noFooter?: boolean;
    disabled?: boolean;
    centered?: boolean;
}

const CommonModalLayout: FC<CommonModalLayoutProps> = ({
    visible,
    children,
    title,
    buttonText,
    className,
    width,
    onClose,
    onSubmit,
    noFooter,
    disabled,
    centered,
}) => {
    const store = useContext(Store);
    const { ui: { intl } } = store;

    useEffect(() => {
        const onEnter = (e: KeyboardEvent) => {
            if (e.key === 'Enter' && onSubmit) {
                onSubmit();
            }
        };
        if (onSubmit) {
            window.addEventListener('keyup', onEnter);
        }
        return () => {
            window.removeEventListener('keyup', onEnter);
        };
    }, [onSubmit]);
    const footer = noFooter ? null : [
        <Button
            type="primary"
            size="large"
            key="submit"
            htmlType="submit"
            onClick={onSubmit}
            disabled={disabled}
        >
            {buttonText}
        </Button>,
        <Button
            type="link"
            size="large"
            key="cancel"
            onClick={onClose}
        >
            {intl.getMessage('cancel')}
        </Button>,
    ];

    return (
        <Modal
            visible={visible}
            title={title}
            wrapClassName={cn('modal', className)}
            onCancel={onClose}
            footer={footer}
            closeIcon={<Icon icon="close_big" />}
            width={width || 480}
            centered={centered}
        >
            {children}
        </Modal>
    );
};

export default CommonModalLayout;
