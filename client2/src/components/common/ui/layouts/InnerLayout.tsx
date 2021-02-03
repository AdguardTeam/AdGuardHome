import { Layout } from 'antd';
import React, { FC } from 'react';
import cn from 'classnames';

import theme from 'Lib/theme';

interface InnerLayoutProps {
    title: string;
    className?: string;
    containerClassName?: string;
}

const InnerLayout: FC<InnerLayoutProps> = ({
    children, title, className, containerClassName,
}) => {
    return (
        <Layout
            className={cn(
                theme.content.content,
                theme.content.content_inner,
                className,
            )}
        >
            <div
                className={cn(
                    theme.content.container,
                    containerClassName,
                )}
            >
                <div className={theme.content.header}>
                    <div className={theme.content.title}>
                        {title}
                    </div>
                </div>
                {children}
            </div>
        </Layout>
    );
};

export default InnerLayout;
