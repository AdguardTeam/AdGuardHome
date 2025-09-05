import React, { ReactNode, useState } from 'react';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import s from './Accordion.module.pcss';

type Props = {
    title: ReactNode;
    children: ReactNode;
    defaultOpen?: boolean;
    className?: string;
};

export const Accordion = ({ title, children, defaultOpen = false, className }: Props) => {
    const [isOpen, setIsOpen] = useState(defaultOpen);

    const toggleOpen = () => {
        setIsOpen(!isOpen);
    };

    return (
        <div className={cn(s.accordion, className)}>
            <button
                type="button"
                className={s.header}
                onClick={toggleOpen}
                aria-expanded={isOpen}
                aria-controls="accordion-content">
                <Icon icon="arrow_bottom" className={cn(s.arrow, { [s.arrowOpen]: isOpen })} />
                <span className={cn(s.title, theme.text.t2, theme.text.semibold)}>{title}</span>
            </button>

            {isOpen && <div id="accordion-content">{children}</div>}
        </div>
    );
};
