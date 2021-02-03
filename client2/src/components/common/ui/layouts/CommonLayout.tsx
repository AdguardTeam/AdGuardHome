import { Layout } from 'antd';
import React, { FC } from 'react';

interface CommonLayoutProps {
    className?: string;
}

const CommonLayout: FC<CommonLayoutProps> = ({ children, className }) => {
    return (
        <Layout className={className}>
            {children}
        </Layout>
    );
};

export default CommonLayout;
