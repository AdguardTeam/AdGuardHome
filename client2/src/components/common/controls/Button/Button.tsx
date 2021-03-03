import React, { FC, FocusEvent } from 'react';
import { Button as ButtonControl } from 'antd';
import cn from 'classnames';

type ButtonSize = 'small' | 'medium' | 'big';
type ButtonType = 'primary' | 'icon' | 'link' | 'outlined' | 'border' | 'ghost' | 'input' | 'edit';
type ButtonHTMLType = 'submit' | 'button' | 'reset';
type ButtonShape = 'circle' | 'round';

export interface ButtonProps {
    className?: string;
    danger?: boolean;
    dataAttrs?: {
        [key: string]: string;
    };
    disabled?: boolean;
    htmlType?: ButtonHTMLType;
    // icon?: IconType | 'dots_loader';
    iconClassName?: string;
    id?: string;
    inGroup?: boolean;
    onClick?: React.MouseEventHandler<HTMLElement>;
    onBlur?: (e: FocusEvent<HTMLInputElement>) => void;
    shape?: ButtonShape;
    size?: ButtonSize;
    type: ButtonType;
    block?: boolean;
}

const Button: FC<ButtonProps> = ({
    children,
    className,
    danger,
    dataAttrs,
    disabled,
    htmlType,
    // icon,
    id,
    onClick,
    onBlur,
    shape,
}) => {
    const buttonClass = cn(
        className,
    );

    return (
        <ButtonControl
            className={buttonClass}
            danger={danger}
            disabled={disabled}
            {...dataAttrs}
            htmlType={htmlType}
            // icon={icon && (icon === 'dots_loader'
            // ? <Dots className={iconClassName} />
            // : <Icon icon={icon} className={iconClassName} />)}
            id={id}
            onClick={onClick}
            onBlur={onBlur}
            shape={shape}
        >
            {children}
        </ButtonControl>
    );
};

export default Button;
