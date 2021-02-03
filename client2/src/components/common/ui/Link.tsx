import React, { FC, MouseEvent } from 'react';
import { Link as L, LinkProps as LProps } from 'react-router-dom';
import cn from 'classnames';

import { linkPathBuilder, RoutePath, LinkParams, LinkParamsKeys } from 'Paths';

interface LinkProps {
    to: RoutePath;
    props?: LinkParams;
    className?: string;
    type?: LProps['type'];
    stop?: boolean;
    disabled?: boolean;
    onClick?: () => void;
    id?: string;
}

const Link: FC<LinkProps> = ({
    to, children, className, props, type, stop, disabled, onClick, id,
}) => {
    if (props) {
        Object.keys(props).forEach((key: unknown) => {
            if (!props[key as LinkParamsKeys]) {
                throw new Error(`Got wrong ${key} propKey: ${props[key as LinkParamsKeys]} in Link`);
            }
        });
    }

    const handleClick = (e: MouseEvent) => {
        if (stop) {
            e.stopPropagation();
        }
        if (onClick) {
            onClick();
        }
    };

    if (disabled) {
        return (
            <div
                id={id}
                tabIndex={0}
                className={cn(className)}
            >
                {children}
            </div>
        );
    }

    return (
        <L
            id={id}
            className={className}
            type={type}
            to={linkPathBuilder(to, props)}
            onClick={handleClick}
        >
            {children}
        </L>
    );
};

export default Link;
