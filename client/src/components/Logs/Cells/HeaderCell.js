import classNames from 'classnames';
import React from 'react';
import propTypes from 'prop-types';
import { useTranslation } from 'react-i18next';

const HeaderCell = ({ content, className }, idx) => {
    const { t } = useTranslation();
    return <div
            key={idx}
            className={classNames('logs__cell--header__item logs__cell logs__text--bold', className)}
            role="columnheader"
    >
        {typeof content === 'string' ? t(content) : content}
    </div>;
};

HeaderCell.propTypes = {
    content: propTypes.oneOfType([propTypes.string, propTypes.element]).isRequired,
    className: propTypes.string,
};

export default HeaderCell;
