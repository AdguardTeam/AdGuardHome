import React, { Component, ReactNode } from 'react';
import cn from 'classnames';

import s from './Errors.module.pcss';

export default class ErrorBoundary extends Component {
    state = {
        isError: false,
    };

    static getDerivedStateFromError(): { isError: boolean } {
        return { isError: true };
    }

    render(): ReactNode {
        const { isError } = this.state;
        const { children } = this.props;

        if (isError) {
            return (
                <div className={cn(s.content, s.content_boundary)}>
                    <div className={s.title}>
                        Something went wrong
                    </div>
                </div>
            );
        }

        return children;
    }
}
