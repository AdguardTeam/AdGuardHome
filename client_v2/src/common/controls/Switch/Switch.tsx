import React, { ChangeEvent, ReactNode, forwardRef, ForwardedRef } from 'react';
import cn from 'clsx';

import s from './Switch.module.pcss';

type Props = {
    id: string;
    checked: boolean;
    disabled?: boolean;
    labelClassName?: string;
    className?: string;
    wrapperClassName?: string;
    onChange: (e: ChangeEvent<HTMLInputElement>) => void;
    children?: ReactNode;
};

export const Switch = forwardRef(
    (
        { id, checked, disabled, children, className, labelClassName, wrapperClassName, onChange }: Props,
        ref: ForwardedRef<HTMLInputElement>,
    ) => {
        const switchControls = (
            <>
                <input
                    id={id}
                    type="checkbox"
                    className={s.input}
                    onChange={onChange}
                    checked={checked}
                    disabled={disabled}
                    ref={ref}
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
    },
);

Switch.displayName = 'Switch';
