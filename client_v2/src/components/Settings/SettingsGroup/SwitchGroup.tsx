import React, { ReactNode, useRef } from 'react';
import cn from 'clsx';

import { Switch } from 'panel/common/controls/Switch';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

type Props = {
    title: string;
    description?: string;
    id: string;
    className?: string;
    checked: boolean;
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    disabled?: boolean;
    children?: ReactNode;
};

export const SwitchGroup = ({ title, description, id, className, checked, onChange, disabled, children }: Props) => {
    const inputRef = useRef<HTMLInputElement>(null);

    const handleRowClick = () => {
        if (disabled || !inputRef.current) {
            return;
        }

        inputRef.current?.click();
    };

    return (
        <div className={cn(s.switch, className)}>
            <div className={s.row} onClick={handleRowClick}>
                <div className={s.text}>
                    <div className={cn(theme.text.t2, theme.text.semibold, s.title)}>{title}</div>
                    {description && <div className={cn(theme.text.t3, s.desc)}>{description}</div>}
                </div>
                <div className={s.input} onClick={(e) => e.stopPropagation()}>
                    <Switch id={id} checked={checked} onChange={onChange} disabled={disabled} ref={inputRef} />
                </div>
            </div>

            {children && <div className={s.content}>{children}</div>}
        </div>
    );
};
