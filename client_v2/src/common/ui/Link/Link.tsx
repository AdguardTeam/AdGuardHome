import React, { MouseEvent, MouseEventHandler, ReactNode } from 'react';
import { Link as L, LinkProps as LProps } from 'react-router-dom';
import cn from 'clsx';
import { LinkParams, linkPathBuilder, RoutePath } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

type Props = {
    to: RoutePath;
    props?: LinkParams;
    className?: string;
    type?: LProps['type'];
    stop?: boolean;
    disabled?: boolean;
    onClick?: MouseEventHandler;
    id?: string;
    query?: Record<string, string | number | boolean>;
    children?: ReactNode;
};

export const Link = ({ to, children, className, props, type, stop, disabled, onClick, id, query }: Props) => {
    if (props) {
        Object.keys(props).forEach((key: string) => {
            if (!props[key]) {
                throw new Error(`Wrong key value: ${key} for route: ${to}`);
            }
        });
    }

    const handleClick = (e: MouseEvent) => {
        // setTimeout fixes bug in safari
        // not scrolling to top
        setTimeout(() => {
            window.scrollTo({ top: 0 });
        }, 100);

        if (stop) {
            e.stopPropagation();
        }
        if (onClick) {
            onClick(e);
        }
    };

    if (disabled) {
        return (
            <div id={id} tabIndex={0} className={cn(className)}>
                {children}
            </div>
        );
    }

    return (
        <L
            id={id}
            className={cn(theme.link.link, className)}
            type={type}
            to={linkPathBuilder(to, props, query)}
            onClick={handleClick}>
            {children}
        </L>
    );
};
