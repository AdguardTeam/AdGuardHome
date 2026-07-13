import { For } from 'solid-js';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { type RoutePathKey } from 'panel/components/Routes/Paths';
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

export const Breadcrumbs = (props: Props) => (
    <div class={s.wrapper}>
        <For each={props.parentLinks}>
            {({ path, title, props: linkProps }) => (
                <div class={s.link}>
                    <Link
                        to={path}
                        class={cn(
                            theme.link.link,
                            theme.link.noDecoration,
                            theme.common.textOverflow,
                        )}
                        props={linkProps}
                    >
                        {title}
                    </Link>
                    <Icon icon="arrow_bottom" class={s.arrow} />
                </div>
            )}
        </For>
        <div class={cn(theme.common.textOverflow, s.current)} title={props.currentTitle}>
            {props.currentTitle}
        </div>
    </div>
);
