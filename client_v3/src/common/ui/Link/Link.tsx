import { type JSX, Show, untrack } from 'solid-js';
import { A } from '@solidjs/router';
import cn from 'clsx';
import { LinkParams, linkPathBuilder, RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

type Props = {
    to: RoutePathKey;
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
    const propsUntracked = untrack(() => linkProps);
    if (propsUntracked.props) {
        Object.keys(propsUntracked.props).forEach((key: string) => {
            if (!(propsUntracked.props as any)[key]) {
                throw new Error(`Wrong key value: ${key} for route: ${propsUntracked.to}`);
            }
        });
    }

    const handleClick = (e: MouseEvent) => {
        setTimeout(() => {
            window.scrollTo({ top: 0 });
        }, 100);

        if (propsUntracked.stop) {
            e.stopPropagation();
        }
        if (propsUntracked.onClick) {
            (propsUntracked.onClick as any)(e);
        }
    };

    return (
        <Show
            when={!propsUntracked.disabled}
            fallback={
                <div id={propsUntracked.id} tabIndex={0} class={cn(propsUntracked.class)}>
                    {propsUntracked.children}
                </div>
            }
        >
            <A
                id={propsUntracked.id}
                class={cn(theme.link.link, propsUntracked.class)}
                href={linkPathBuilder(
                    propsUntracked.to,
                    propsUntracked.props,
                    propsUntracked.query,
                )}
                onClick={handleClick}
            >
                {propsUntracked.children}
            </A>
        </Show>
    );
};
