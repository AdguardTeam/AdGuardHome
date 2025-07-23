import React from 'react';
import cn from 'clsx';
import { Icon, Link } from 'panel/common/ui';
import { RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

type Props = {
    parentLinks: {
        path: RoutePathKey;
        title: string;
        props?: Partial<Record<string, string | number>>;
    }[];
    currentTitle: string;
};

export const Breadcrumbs = ({ parentLinks, currentTitle }: Props) => (
    <div className={s.wrapper}>
        {parentLinks.map(({ path, title, props }) => (
            <div key={path} className={s.link}>
                <Link
                    to={path}
                    className={cn(theme.link.link, theme.link.no_decoration, theme.common.textOverflow)}
                    props={props}>
                    {title}
                </Link>
                <Icon icon="arrow_bottom" className={s.arrow} />
            </div>
        ))}
        <div className={theme.common.textOverflow}>{currentTitle}</div>
    </div>
);
