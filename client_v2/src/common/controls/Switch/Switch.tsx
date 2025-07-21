import React, { ChangeEvent, ReactNode } from 'react';
import cn from 'clsx';

import s from './Switch.module.pcss';

type Props = {
    id: string;
    checked: boolean;
    disabled?: boolean;
    labelClassName?: string;
    className?: string;
    wrapperClassName?: string;
    handleChange: (e: ChangeEvent<HTMLInputElement>) => void;
    children?: ReactNode;
};

export const Switch = ({
    id,
    checked,
    disabled,
    children,
    className,
    labelClassName,
    wrapperClassName,
    handleChange,
}: Props) => {
    const switchControls = (
        <>
            <input
                id={id}
                type="checkbox"
                className={s.input}
                onChange={handleChange}
                checked={checked}
                disabled={disabled}
            />
            <div className={s.handler} />
            {children && <div className={cn(s.label, labelClassName)}>{children}</div>}
        </>
    );

    const getContent = () => {
        if (wrapperClassName) {
            return <div className={wrapperClassName}>{switchControls}</div>;
        }

        return switchControls;
    };

    return (
        <label htmlFor={id} className={cn(s.switch, className, { [s.disabled]: disabled })}>
            {getContent()}
        </label>
    );
};

export default Switch;
