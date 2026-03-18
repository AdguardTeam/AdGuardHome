import React, { ReactNode } from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';

import s from './DetailModal.module.pcss';

type Props = {
    label: string;
    value: ReactNode;
};

const hasValue = (value: ReactNode) => value !== undefined && value !== null && value !== '' && value !== false;

export const DetailRow = ({ label, value }: Props) => {
    if (!hasValue(value)) {
        return null;
    }

    return (
        <div className={cn(s.row, theme.text.t3)}>
            <span className={cn(s.label, theme.text.semibold)}>{label}</span>
            {typeof value === 'string' || typeof value === 'number'
                ? <span className={s.value}>{value}</span>
                : value}
        </div>
    );
};
