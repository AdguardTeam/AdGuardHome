import React, { useEffect, useState } from 'react';
import cn from 'clsx';
import { Icon, IconType } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

type AccordionItem = {
    label: string;
    path: string;
    routePath: RoutePathKey;
};

type Props = {
    title: string;
    icon: IconType;
    items: AccordionItem[];
    isActive: (path: string | string[], full?: boolean) => boolean;
    className?: string;
};

export const AccordionSection = ({ title, icon, items, isActive, className }: Props) => {
    const [isOpen, setIsOpen] = useState(false);

    const isAnyItemActive = items.some((item) => isActive(item.path));

    useEffect(() => {
        if (!isOpen && isAnyItemActive) {
            setIsOpen(true);
        }
    }, []);

    const toggleAccordion = (e: React.MouseEvent) => {
        e.stopPropagation();
        setIsOpen(!isOpen);
    };

    return (
        <div className={cn(s.menuLinkWrapper, className)}>
            <div
                className={cn(s.menuLink, s.accordionHeader, {
                    [s.activeLink]: isAnyItemActive,
                    [s.accordionOpen]: isOpen,
                })}
                onClick={toggleAccordion}
                role="button"
                tabIndex={0}>
                <Icon className={s.linkIcon} icon={icon} />
                <span className={theme.common.textOverflow}>{title}</span>
                <Icon className={cn(s.accordionArrow, { [s.accordionArrowOpen]: isOpen })} icon="arrow_bottom" />
            </div>
            {isOpen && (
                <div className={s.accordionContent}>
                    {items.map((item) => (
                        <div key={item.path} className={s.accordionItem}>
                            <Link
                                className={cn(s.accordionLink, {
                                    [s.activeLink]: isActive(item.path),
                                })}
                                to={item.routePath}>
                                <span className={theme.common.textOverflow}>{item.label}</span>
                            </Link>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};
