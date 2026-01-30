import React, { ReactNode, useEffect, useRef, useState } from 'react';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import RCDropdown from 'rc-dropdown';

import './Dropdown.pcss';
import s from './Dropdown.module.pcss';

const TIMEOUT_HIDE_TOOLTIP = 1000;

type Props = {
    overlayClassName?: string;
    menu: Parameters<typeof RCDropdown>[0]['overlay'];
    position?: Parameters<typeof RCDropdown>[0]['placement'];
    trigger: 'click' | 'hover';
    noIcon?: true;
    iconClassName?: string;
    className?: string;
    openClassName?: string;
    open?: boolean;
    onOpenChange?: (e: boolean) => void;
    widthAuto?: boolean;
    flex?: boolean;
    minOverlayWidthMatchTrigger?: boolean;
    flexWrapper?: boolean;
    childrenClassName?: string;
    wrapClassName?: string;
    children?: ReactNode;
    isSelect?: boolean;
    disableAnimation?: boolean;
    disabled?: boolean;
    autoClose?: boolean;
};

export const Dropdown = ({
    children,
    open,
    onOpenChange,
    overlayClassName,
    className,
    menu,
    position,
    trigger,
    noIcon,
    iconClassName,
    widthAuto,
    flex,
    openClassName,
    childrenClassName,
    minOverlayWidthMatchTrigger,
    flexWrapper,
    wrapClassName,
    isSelect,
    disableAnimation,
    disabled,
    autoClose = false,
}: Props) => {
    const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
    const [visible, setVisible] = useState(!!open);
    const onVisibleChange = (e: boolean) => {
        if (disabled) {
            return;
        }

        if (onOpenChange) {
            onOpenChange(e);
        }
        setVisible(e);
    };

    useEffect(() => {
        if (typeof open === 'boolean') {
            setVisible(open);
        }

        return () => {
            setVisible(false);
        };
    }, [open]);

    const handleOverlayClick = () => {
        if (!autoClose) {
            return;
        }

        if (timer.current) {
            clearTimeout(timer.current);
        }
        timer.current = setTimeout(() => {
            setVisible(false);
        }, TIMEOUT_HIDE_TOOLTIP);
    };

    return (
        <RCDropdown
            overlayClassName={cn(s.overlay, overlayClassName, {
                [s.widthAuto]: widthAuto,
                [s.selectOverlay]: isSelect,
            })}
            trigger={trigger}
            mouseEnterDelay={trigger === 'hover' ? 0.2 : 0}
            overlay={menu}
            animation={disableAnimation ? undefined : 'slide-up'}
            placement={position}
            visible={visible}
            onVisibleChange={onVisibleChange}
            minOverlayWidthMatchTrigger={!!minOverlayWidthMatchTrigger}
            onOverlayClick={handleOverlayClick}>
            <div
                className={cn(
                    className,
                    s.wrapper,
                    {
                        [s.open]: flex,
                        [s.disabled]: disabled,
                    },
                    visible && openClassName ? openClassName : null,
                    wrapClassName,
                )}>
                <div className={cn(childrenClassName, { [s.wrapper]: flexWrapper })} onClick={handleOverlayClick}>
                    {children}
                </div>
                {!noIcon && (
                    <Icon className={cn(s.arrow, iconClassName, { [s.active]: visible })} icon="arrow_bottom" />
                )}
            </div>
        </RCDropdown>
    );
};
