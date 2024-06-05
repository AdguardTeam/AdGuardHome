import React from 'react';
import classNames from 'classnames';
import { useTranslation } from 'react-i18next';
import './Loading.css';

interface LoadingProps {
    className?: string;
    text?: string;
}

const Loading = ({ className, text }: LoadingProps) => {
    const { t } = useTranslation();

    return <div className={classNames('loading', className)}>{t(text)}</div>;
};

export default Loading;
