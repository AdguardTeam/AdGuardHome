import { type JSX, Show, untrack } from 'solid-js';
import { A } from '@solidjs/router';
import cn from 'clsx';
import {
    LinkParams,
    linkPathBuilder,
    RoutePathKey,
    SCROLL_QUERY_KEY,
} from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

type Props = {
    to: RoutePathKey;
    hash?: string;
    props?: LinkParams;
    class?: string;
    type?: string;
    stop?: boolean;
    disabled?: boolean;
    onClick?: JSX.EventHandler<HTMLAnchorElement, MouseEvent>;
    id?: string;
    query?: Record<string, string | number | boolean>;
    children?: JSX.Element;
};

export const Link = (linkProps: Props) => {
    untrack(() => {
        if (linkProps.props) {
            Object.keys(linkProps.props).forEach((key: string) => {
                if (!(linkProps.props as any)[key]) {
                    throw new Error(`Wrong key value: ${key} for route: ${linkProps.to}`);
                }
            });
        }
    });

    const handleClick = (e: MouseEvent) => {
        // Don't scroll to top when navigating to a specific section (scroll handled by the target page)
        if (!linkProps.query?.[SCROLL_QUERY_KEY]) {
            setTimeout(() => {
                window.scrollTo({ top: 0 });
            }, 100);
        }

        if (linkProps.stop) {
            e.stopPropagation();
        }
        if (linkProps.onClick) {
            (linkProps.onClick as any)(e);
        }
    };

    return (
        <Show
            when={!linkProps.disabled}
            fallback={
                <div id={linkProps.id} tabIndex={0} class={cn(linkProps.class)}>
                    {linkProps.children}
                </div>
            }
        >
            <A
                id={linkProps.id}
                class={cn(theme.link.link, linkProps.class)}
                href={linkPathBuilder(
                    linkProps.to,
                    linkProps.props,
                    linkProps.query,
                    linkProps.hash,
                )}
                onClick={handleClick}
            >
                {linkProps.children}
            </A>
        </Show>
    );
};
