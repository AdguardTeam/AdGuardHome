import React from 'react';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

export type BreadcrumbLink = {
    path: RoutePathKey;
    title: string;
    props?: Partial<Record<string, string | number>>;
};

type Props = {
    parentLinks: BreadcrumbLink[];
    currentTitle: string;
};

export const Breadcrumbs = ({ parentLinks, currentTitle }: Props) => (
    <div className={s.wrapper}>
        {parentLinks.map(({ path, title, props }) => (
            <div key={path} className={s.link}>
                <Link
                    to={path}
                    className={cn(
                        theme.link.link,
                        theme.link.noDecoration,
                        theme.common.textOverflow,
                    )}
                    props={props}
                >
                    {title}
                </Link>
                <Icon icon="arrow_bottom" className={s.arrow} />
            </div>
        ))}
        <div className={cn(theme.common.textOverflow, s.current)} title={currentTitle}>
            {currentTitle}
        </div>
    </div>
);
