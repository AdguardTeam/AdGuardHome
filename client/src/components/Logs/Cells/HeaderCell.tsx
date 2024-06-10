import classNames from 'classnames';
import React from 'react';
import { useTranslation } from 'react-i18next';

interface HeaderCellProps {
    content: string | React.ReactElement;
    className?: string;
}

const HeaderCell = ({ content, className }: HeaderCellProps, idx: any) => {
    const { t } = useTranslation();

    return (
        <div
            key={idx}
            className={classNames('logs__cell--header__item logs__cell logs__text--bold', className)}
            role="columnheader">
            {typeof content === 'string' ? t(content) : content}
        </div>
    );
};

export default HeaderCell;
