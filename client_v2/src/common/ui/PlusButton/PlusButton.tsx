import React from 'react';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import s from './PlusButton.module.pcss';

type Props = {
    children: React.ReactNode;
    className?: string;
    onClick: () => void;
    disabled?: boolean;
    testId?: string;
};

export const PlusButton = ({ children, className, onClick, disabled, testId }: Props) => (
    <button
        type="button"
        className={cn(s.plusButton, className)}
        onClick={onClick}
        disabled={disabled}
        data-testid={testId}
    >
        <Icon icon="plus" />
        {children}
    </button>
);
